<!--

  You can edit the file as you like before or after the HTML comment,
  but do not edit the API documentation between the following HTML comments,
  it was automatically generated from the index.d.ts file.

  You can regenerate the API documentation and bindings code at any time
  by "go generate ." command. The "//go:generate ..." comments required for this
  can be found in the environment.go file.

-->

**xk6-environment** provides a way to simplify definition of a k6 test with environment attached. This allows to bootstrap and execute the test within specific environment in a repeatable fashion while keeping the environment isolated.

> [!WARNING] 
> xk6-environment is an experimental extension. Consider limitations below and existing GitHub issues before usage.

Some use cases for xk6-environment:
- As a first-class object, the environment itself can be viewed as a SUT (system under test).
  - Answer questions like: does my environment reach a state X under conditions Y?
- Repeatable execution of testing logic within a given environment, without resorting to additional Bash scripts around k6 invocation.
- Infrastructure tests or experiments.
- A way to share a k6 test with environment attached. E.g. a k6 test that illustrates an issue that happens only with a certain version of Prometheus deployment.

Current implementation is done via [vcluster](https://www.vcluster.com/) tool which creates a virtual, "ephemeral" cluster within existing Kubernetes cluster.

## Prerequisites

You need:
- Access to Kubernetes cluster via `kubectl`.
- Install [vcluster CLI](https://www.vcluster.com/docs/getting-started/setup).
    - this requirement might be lifted in the future, see the [issue](https://github.com/grafana/xk6-environment/issues/1).

The simplest way to get started with xk6-environment:

```bash
make build
./k6 run example/sample.js
```

## How it works

xk6-environment creates an environment by starting a new, virtual cluster within the Kubernetes cluster you're currently connected to.

Optionally, a user can configure a folder with Kubernetes manifests which describes initial deployments within the environment. The extension doesn't check for correctness or validation anywhere: it will try to deploy and will report an error if it's not possible. In case of complex deployments, one can add a `kustomization.yaml` file to the folder: it will be located by xk6-environment and processed correctly. The easiest way to add a `kustomization.yaml` is to run the following command from your folder with manifests:

```bash
kustomize create --autodetect --recursive .
```

At the end of the test the environment can be deleted which triggers removal of virtual cluster as well.

The basic workflow looks as follows:

```js
import { Environment } from 'k6/x/environment';
import { sleep } from 'k6';

export const options = {
  setupTimeout: '60s',
};

const PARENT = "..."

const env = new Environment({
  name: "my-test-environment",
  implementation: "vcluster",
  initFolder: PARENT, // initial folder with everything that wil be loaded at init
})

export function setup() {
  console.log("init returns", env.init());

}

export default function () {
  // something meaningful happens here
}

export function teardown() {
  console.log("delete returns", env.delete());
}
```

The rest of the provided methods allow to perform basic Kubenetes functions _within_ the environment.

<!-- More samples can be found here -->

## Limitations

The most important current limitations:
- xk6-environment support only `VU: 1` tests. Given the nature of virtual environments, it is yet unclear if there is a use case that requires more than 1 VU.
- Kubernetes context corresponding to vcluster is created on `create` method but not yet removed on `delete`.
- Since xk6-environment accesses and edits your `KUBECONFIG`, if you tinker with it simultaneously with the test execution, the result might be unpredictable. For example, changing Kubernetes context during execution of xk6-environment test will likely make the test fail.
<!-- begin:api -->
xk6-environment
===============

ˮsummaryˮ

<details><summary><em>Example</em></summary>

```ts
import globalEnvironment, { Environment } from "k6/x/environment"

export default function () {
  console.log(globalEnvironment.greeting)

  let instance = new Environment("Wonderful World")
  console.log(instance.greeting)
}
```

</details>

<details>
<summary><strong>Build</strong></summary>

The [xk6](https://github.com/grafana/xk6) build tool can be used to build a k6 that will include xk6-environment extension:

```bash
$ xk6 build --with github.com/grafana/xk6-environment@latest
```

For more build options and how to use xk6, check out the [xk6 documentation]([xk6](https://github.com/grafana/xk6)).

</details>

API
===

Environment
-----------

This is the primary class of the environment extension.

<details><summary><em>Example</em></summary>

```ts
import { Environment } from "k6/x/environment"

export default function () {
  let env = new Environment({
    name: "my-env",
    implementation: "vcluster",
    initFolder: "my-folder-with-manifests/",
  })
}
```

</details>

### Environment()

```ts
constructor(params: object);
```

-	`name` name of the environment

-	`implementation` implementation for the environment (only "vcluster" for now)

-	`initFolder` optional, a folder containing base manifests to apply on initialization of environment

Defines a new Environment instance.

### Environment.init()

```ts
init();
```

init creates an Environment as defined in constructor.

### Environment.delete()

```ts
delete();
```

delete removes an existing Environment.

### Environment.apply()

```ts
apply(file: string);
```

-	`file` is expected to be a readable yaml file (Kubernetes manifest).

apply reads the contents of the file and applies them to the virtual cluster.

### Environment.applySpec()

```ts
applySpec(spec: string);
```

-	`spec` is expected to be a yaml manifest.

applySpec applies the spec to the virtual cluster.

### Environment.wait()

```ts
wait(condition: object, opts?: object);
```

-	`condition` describes the wait condition itself. It should have name, namespace, kind fields. It can be configured with fields: 1) "reason" to wait for Kubernetes event, 2) "condition\_type" and "value", to wait for `.status.conditions[]`, 3) "status\_key" and "status\_value" to wait for custom `.status` value.

-	`opts` optional configuration of timeout and interval (defaults are 1h and 2s), for how often to perfrom a check of wait condition.

`wait` method blocks execution of the test iteration until a certain condition is reached or until a timeout. There are 3 major types of conditions now:

1.	Wait until a given Kubernetes event.

2.	Wait until a given `.status.conditions[]` reaches a given value.

3.	Wait until a custom field in `.status` reaches a given value.

### Environment.getN()

```ts
getN(type: string, opts?: object): number;
```

-	`type` is a kind of resource (currently only "pods" are supported).

-	`opts` optional parameteters for the resource, like namespace and labels.

getN is a substitute for get(), hopefully temporary. See [tygor's](https://github.com/szkiba/tygor) roadmap about support for arrays.
<!-- end:api -->
