# Credential rotation consumer finalizer guard

OpenStack service operators that consume rotating credential secrets
(transport URL, application credentials, or similar) attach a consumer
finalizer to the old secret so it is not deleted until every sub-service
has rolled out with the new credential.

## Shared helpers

Use helpers from these packages instead of duplicating guard logic in each
operator:

- `object.ManageSecretConsumerFinalizer` — add finalizer to current secret
- `object.RemoveSecretConsumerFinalizer` — remove finalizer from old secret
- `object.FinalizeSecretRotation` — rotation guard: hold old finalizer
  until all sub-services are ready, then release
- `condition.CredentialRotationGuardReady` — compute `guardReady` for the
  rotation guard
- `helper.EnsureFresh` — bypass informer cache during rotation so the
  parent reads accurate sub-CR conditions from the API server
- `helper.SetAPIReader` — configure the cache-bypassing reader
- `statefulset.IsReady` / `deployment.IsReady` — check UpdatedReplicas,
  CurrentRevision, and ObservedGeneration so DeploymentReady is only
  True when all pods have rolled to the new config

## Parent controller integration

### 1. Wire up APIReader

Add `APIReader client.Reader` to the reconciler struct and pass
`mgr.GetAPIReader()` from `cmd/main.go`. In `Reconcile`, call
`helper.SetAPIReader(r.APIReader)` after creating the helper.

### 2. Pass transport URL directly — never through Status

Pass `transportURL.Status.SecretName` as a parameter to sub-CR creation
functions and config generation. Never read from
`instance.Status.TransportURLSecret` when building sub-CR specs — the
status field is only used as the "old" value for `FinalizeSecretRotation`.

Set `instance.Status.TransportURLSecret` early only for first-time setup:

```go
if instance.Status.TransportURLSecret == "" ||
    instance.Status.TransportURLSecret == transportURL.Status.SecretName {
    instance.Status.TransportURLSecret = transportURL.Status.SecretName
}
```

During rotation (old != current), the status is updated solely by
`FinalizeSecretRotation` at the end of reconcile.

### 3. Track stability and call EnsureFresh

Use a simple boolean instead of StabilityTracker:

```go
allSubCRsStable := true
rotationInProgress := instance.Status.TransportURLSecret != "" &&
    instance.Status.TransportURLSecret != transportURL.Status.SecretName

// After each sub-CR CreateOrPatch:
subCR, op, err := r.subCRCreateOrUpdate(ctx, instance, transportURL.Status.SecretName)
if err != nil { return ctrl.Result{}, err }
if err := helper.EnsureFresh(ctx, op, subCR, rotationInProgress); err != nil {
    return ctrl.Result{}, err
}
if op != controllerutil.OperationResultNone {
    allSubCRsStable = false
}
```

`EnsureFresh` re-reads the sub-CR directly from the API server when
`op == None` and rotation is in progress. Without this, the informer
cache may return stale data where `Generation == ObservedGeneration`
from before the spec change, causing the guard to pass prematurely.

### 4. Transport URL annotation (operators without TransportURLSecret in sub-CR spec)

For operators whose sub-CR specs do not include a `TransportURLSecret`
field (e.g. nova, watcher, glance, designate, octavia, telemetry), add
an annotation inside the `CreateOrPatch` mutate function to force a
spec change when the transport URL rotates:

```go
op, err := controllerutil.CreateOrPatch(ctx, r.Client, subCR, func() error {
    subCR.Spec = spec
    if subCR.Annotations == nil {
        subCR.Annotations = map[string]string{}
    }
    subCR.Annotations["openstack.org/transport-url-secret"] = transportURLSecretName
    return controllerutil.SetControllerReference(instance, subCR, r.Scheme)
})
```

This is not needed for operators that pass `TransportURLSecret` directly
in the sub-CR spec (cinder, manila, ironic, barbican, heat) — the spec
field change already makes `CreateOrPatch` return `"updated"`.

### 5. Compute the rotation guard and finalize

```go
guardReady := condition.CredentialRotationGuardReady(
    allSubCRsStable,
    &instance.Status.Conditions,
)

instance.Status.TransportURLSecret, err = object.FinalizeSecretRotation(
    ctx, helper, instance.Namespace,
    instance.Status.TransportURLSecret,
    transportURL.Status.SecretName,
    myTransportConsumerFinalizer,
    guardReady,
)
```

The same pattern works for application credential secrets:

```go
instance.Status.ApplicationCredentialSecret, err = object.FinalizeSecretRotation(
    ctx, helper, instance.Namespace,
    instance.Status.ApplicationCredentialSecret,
    instance.Spec.Auth.ApplicationCredentialSecret,
    myACConsumerFinalizer,
    guardReady,
)
```

## Sub-CR integration

Sub-CR controllers should use `statefulset.IsReady()` or
`deployment.IsReady()` for the `DeploymentReadyCondition` check:

```go
if statefulset.IsReady(ssData) {
    instance.Status.Conditions.MarkTrue(
        condition.DeploymentReadyCondition,
        condition.DeploymentReadyMessage)
} else if *instance.Spec.Replicas > 0 {
    instance.Status.Conditions.Set(condition.FalseCondition(
        condition.DeploymentReadyCondition,
        condition.RequestedReason,
        condition.SeverityInfo,
        condition.DeploymentReadyRunningMessage))
}
```

These helpers check `*Replicas == ReadyReplicas`, `*Replicas == UpdatedReplicas`,
`Generation == ObservedGeneration`, and `CurrentRevision == UpdateRevision`,
ensuring DeploymentReady is only True when all pods have rolled to the
new config.
