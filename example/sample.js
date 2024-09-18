import { Environment } from 'k6/x/environment';
import http from 'k6/http';

const env = new Environment({
  name: "some-name", // name of the test = name of vcluster
  implementation: "vcluster", // k8s is prerequisite
  initFolder: "example/", // initial folder with everything that wil be loaded at init
})

export function setup() {
  // to ensure execution happens only once, we run creation of environment in setup
  console.log("init returns", env.init());
}

export default function () {
  env.apply("example/pod.yaml"); // deploys nginx pod

  env.applySpec(`apiVersion: v1
kind: Pod
metadata:
  name: nginx2
spec:
  containers:
  - name: nginx
    image: nginx:1.14.2
    ports:
      - containerPort: 80
    `);

  // sync function, blocking
  // Wait until nginx Pod generates Kubernetes event "Started"
  let err = env.wait({
    kind: "Pod",
    name: "nginx",
    namespace: "default",
    reason: "Started", // k8s event
  });
  console.log("wait for nginx returns", err);

  // Wait until .status.conditions Ready reaches value True
  // for nginx2 Pod
  console.log("wait for nginx2 returns", env.wait({
    kind: "Pod",
    name: "nginx2",
    namespace: "default",
    condition_type: "Ready",
    value: "True",
  }, {
    timeout: "1m",
  }));
}

export function teardown() {
  console.log("delete returns", env.delete());
}