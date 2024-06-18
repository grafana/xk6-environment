<!--

  You can edit the file as you like before or after the HTML comment,
  but do not edit the API documentation between the following HTML comments,
  it was automatically generated from the index.d.ts file.

  You can regenerate the API documentation and bindings code at any time
  by "go generate ." command. The "//go:generate ..." comments required for this
  can be found in the environment.go file.

-->
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

The [examples](https://github.com/grafana/xk6-environment/blob/master/examples) directory contains examples of how to use the xk6-environment extension. A k6 binary containing the xk6-environment extension is required to run the examples. *If the search path also contains the k6 command, don't forget to specify which k6 you want to run (for example `./k6`\)*.

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
  let instance = new Environment("Wonderful World")
  console.log(instance.greeting)
}
```

</details>

### Environment()

```ts
constructor(params: object);
```

-	`name` name of the environment

-	`type` implementation for the environment (only "vcluster" for now)

-	`initFolder` folder containing base manifests to apply on initialization of environment

Create a new Environment instance.

### Environment.init()

```ts
init();
```

### Environment.delete()

```ts
delete();
```

### Environment.apply()

```ts
apply(file: string);
```

### Environment.applySpec()

```ts
applySpec(spec: string);
```

### Environment.wait()

```ts
wait(condition: object, opts?: object);
```
<!-- end:api -->
