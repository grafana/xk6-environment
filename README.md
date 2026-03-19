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

The rest of the provided methods allow to perform basic Kubernetes functions _within_ the environment. See `index.d.ts` for full description of available API.

<!-- More samples can be found here -->

## Limitations

The most important current limitations:
- xk6-environment support only `VU: 1` tests. Given the nature of virtual environments, it is yet unclear if there is a use case that requires more than 1 VU.
- Kubernetes context corresponding to vcluster is created on `create` method but not yet removed on `delete`.
- Since xk6-environment accesses and edits your `KUBECONFIG`, if you tinker with it simultaneously with the test execution, the result might be unpredictable. For example, changing Kubernetes context during execution of xk6-environment test will likely make the test fail.

