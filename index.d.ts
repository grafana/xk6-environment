/**
 * ˮsummaryˮ
 *
 * @example
 * ```ts
 * import globalEnvironment, { Environment } from "k6/x/environment"
 *
 * export default function () {
 *   console.log(globalEnvironment.greeting)
 *
 *   let instance = new Environment("Wonderful World")
 *   console.log(instance.greeting)
 * }
 * ```
 */
export as namespace environment;

/**
 * This is the primary class of the environment extension.
 *
 * @example
 * ```ts
 * import { Environment } from "k6/x/environment"
 *
 * export default function () {
 *   let instance = new Environment("Wonderful World")
 *   console.log(instance.greeting)
 * }
 * ```
 */
export declare class Environment {
  /**
   * Create a new Environment instance.
   *
   * @param name name of the environment
   * @param type implementation for the environment (only "vcluster" for now)
   * @param initFolder folder containing base manifests to apply on initialization of environment
   */
  constructor(name: string, type: string, initFolder: string);

  init(): void;
  delete(): void;

  // apply(files: string[]); arrays are not supported by Tygor :sweat_smile:
  apply(file: string): void;
  applySpec(spec: string): void; // we have to use a diff namehere: method overload is not supported

  wait(obj: any): void;

  // TODO:
  // list(resource: string, namespace: string);
  // delete();
}

/** Default Environment instance. */
declare const defaultEnvironment: Environment;

/** Default Environment instance. */
export default defaultEnvironment;
