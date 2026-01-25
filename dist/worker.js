var __defProp = Object.defineProperty;
var __name = (target, value) => __defProp(target, "name", { value, configurable: true });

// node_modules/unenv/dist/runtime/_internal/utils.mjs
// @__NO_SIDE_EFFECTS__
function createNotImplementedError(name) {
  return new Error(`[unenv] ${name} is not implemented yet!`);
}
__name(createNotImplementedError, "createNotImplementedError");
// @__NO_SIDE_EFFECTS__
function notImplemented(name) {
  const fn = /* @__PURE__ */ __name(() => {
    throw /* @__PURE__ */ createNotImplementedError(name);
  }, "fn");
  return Object.assign(fn, { __unenv__: true });
}
__name(notImplemented, "notImplemented");
// @__NO_SIDE_EFFECTS__
function notImplementedClass(name) {
  return class {
    __unenv__ = true;
    constructor() {
      throw new Error(`[unenv] ${name} is not implemented yet!`);
    }
  };
}
__name(notImplementedClass, "notImplementedClass");

// node_modules/unenv/dist/runtime/node/internal/perf_hooks/performance.mjs
var _timeOrigin = globalThis.performance?.timeOrigin ?? Date.now();
var _performanceNow = globalThis.performance?.now ? globalThis.performance.now.bind(globalThis.performance) : () => Date.now() - _timeOrigin;
var nodeTiming = {
  name: "node",
  entryType: "node",
  startTime: 0,
  duration: 0,
  nodeStart: 0,
  v8Start: 0,
  bootstrapComplete: 0,
  environment: 0,
  loopStart: 0,
  loopExit: 0,
  idleTime: 0,
  uvMetricsInfo: {
    loopCount: 0,
    events: 0,
    eventsWaiting: 0
  },
  detail: void 0,
  toJSON() {
    return this;
  }
};
var PerformanceEntry = class {
  static {
    __name(this, "PerformanceEntry");
  }
  __unenv__ = true;
  detail;
  entryType = "event";
  name;
  startTime;
  constructor(name, options) {
    this.name = name;
    this.startTime = options?.startTime || _performanceNow();
    this.detail = options?.detail;
  }
  get duration() {
    return _performanceNow() - this.startTime;
  }
  toJSON() {
    return {
      name: this.name,
      entryType: this.entryType,
      startTime: this.startTime,
      duration: this.duration,
      detail: this.detail
    };
  }
};
var PerformanceMark = class PerformanceMark2 extends PerformanceEntry {
  static {
    __name(this, "PerformanceMark");
  }
  entryType = "mark";
  constructor() {
    super(...arguments);
  }
  get duration() {
    return 0;
  }
};
var PerformanceMeasure = class extends PerformanceEntry {
  static {
    __name(this, "PerformanceMeasure");
  }
  entryType = "measure";
};
var PerformanceResourceTiming = class extends PerformanceEntry {
  static {
    __name(this, "PerformanceResourceTiming");
  }
  entryType = "resource";
  serverTiming = [];
  connectEnd = 0;
  connectStart = 0;
  decodedBodySize = 0;
  domainLookupEnd = 0;
  domainLookupStart = 0;
  encodedBodySize = 0;
  fetchStart = 0;
  initiatorType = "";
  name = "";
  nextHopProtocol = "";
  redirectEnd = 0;
  redirectStart = 0;
  requestStart = 0;
  responseEnd = 0;
  responseStart = 0;
  secureConnectionStart = 0;
  startTime = 0;
  transferSize = 0;
  workerStart = 0;
  responseStatus = 0;
};
var PerformanceObserverEntryList = class {
  static {
    __name(this, "PerformanceObserverEntryList");
  }
  __unenv__ = true;
  getEntries() {
    return [];
  }
  getEntriesByName(_name, _type) {
    return [];
  }
  getEntriesByType(type) {
    return [];
  }
};
var Performance = class {
  static {
    __name(this, "Performance");
  }
  __unenv__ = true;
  timeOrigin = _timeOrigin;
  eventCounts = /* @__PURE__ */ new Map();
  _entries = [];
  _resourceTimingBufferSize = 0;
  navigation = void 0;
  timing = void 0;
  timerify(_fn, _options) {
    throw createNotImplementedError("Performance.timerify");
  }
  get nodeTiming() {
    return nodeTiming;
  }
  eventLoopUtilization() {
    return {};
  }
  markResourceTiming() {
    return new PerformanceResourceTiming("");
  }
  onresourcetimingbufferfull = null;
  now() {
    if (this.timeOrigin === _timeOrigin) {
      return _performanceNow();
    }
    return Date.now() - this.timeOrigin;
  }
  clearMarks(markName) {
    this._entries = markName ? this._entries.filter((e3) => e3.name !== markName) : this._entries.filter((e3) => e3.entryType !== "mark");
  }
  clearMeasures(measureName) {
    this._entries = measureName ? this._entries.filter((e3) => e3.name !== measureName) : this._entries.filter((e3) => e3.entryType !== "measure");
  }
  clearResourceTimings() {
    this._entries = this._entries.filter((e3) => e3.entryType !== "resource" || e3.entryType !== "navigation");
  }
  getEntries() {
    return this._entries;
  }
  getEntriesByName(name, type) {
    return this._entries.filter((e3) => e3.name === name && (!type || e3.entryType === type));
  }
  getEntriesByType(type) {
    return this._entries.filter((e3) => e3.entryType === type);
  }
  mark(name, options) {
    const entry = new PerformanceMark(name, options);
    this._entries.push(entry);
    return entry;
  }
  measure(measureName, startOrMeasureOptions, endMark) {
    let start;
    let end;
    if (typeof startOrMeasureOptions === "string") {
      start = this.getEntriesByName(startOrMeasureOptions, "mark")[0]?.startTime;
      end = this.getEntriesByName(endMark, "mark")[0]?.startTime;
    } else {
      start = Number.parseFloat(startOrMeasureOptions?.start) || this.now();
      end = Number.parseFloat(startOrMeasureOptions?.end) || this.now();
    }
    const entry = new PerformanceMeasure(measureName, {
      startTime: start,
      detail: {
        start,
        end
      }
    });
    this._entries.push(entry);
    return entry;
  }
  setResourceTimingBufferSize(maxSize) {
    this._resourceTimingBufferSize = maxSize;
  }
  addEventListener(type, listener, options) {
    throw createNotImplementedError("Performance.addEventListener");
  }
  removeEventListener(type, listener, options) {
    throw createNotImplementedError("Performance.removeEventListener");
  }
  dispatchEvent(event) {
    throw createNotImplementedError("Performance.dispatchEvent");
  }
  toJSON() {
    return this;
  }
};
var PerformanceObserver = class {
  static {
    __name(this, "PerformanceObserver");
  }
  __unenv__ = true;
  static supportedEntryTypes = [];
  _callback = null;
  constructor(callback) {
    this._callback = callback;
  }
  takeRecords() {
    return [];
  }
  disconnect() {
    throw createNotImplementedError("PerformanceObserver.disconnect");
  }
  observe(options) {
    throw createNotImplementedError("PerformanceObserver.observe");
  }
  bind(fn) {
    return fn;
  }
  runInAsyncScope(fn, thisArg, ...args) {
    return fn.call(thisArg, ...args);
  }
  asyncId() {
    return 0;
  }
  triggerAsyncId() {
    return 0;
  }
  emitDestroy() {
    return this;
  }
};
var performance = globalThis.performance && "addEventListener" in globalThis.performance ? globalThis.performance : new Performance();

// node_modules/@cloudflare/unenv-preset/dist/runtime/polyfill/performance.mjs
globalThis.performance = performance;
globalThis.Performance = Performance;
globalThis.PerformanceEntry = PerformanceEntry;
globalThis.PerformanceMark = PerformanceMark;
globalThis.PerformanceMeasure = PerformanceMeasure;
globalThis.PerformanceObserver = PerformanceObserver;
globalThis.PerformanceObserverEntryList = PerformanceObserverEntryList;
globalThis.PerformanceResourceTiming = PerformanceResourceTiming;

// node_modules/unenv/dist/runtime/node/console.mjs
import { Writable } from "node:stream";

// node_modules/unenv/dist/runtime/mock/noop.mjs
var noop_default = Object.assign(() => {
}, { __unenv__: true });

// node_modules/unenv/dist/runtime/node/console.mjs
var _console = globalThis.console;
var _ignoreErrors = true;
var _stderr = new Writable();
var _stdout = new Writable();
var log = _console?.log ?? noop_default;
var info = _console?.info ?? log;
var trace = _console?.trace ?? info;
var debug = _console?.debug ?? log;
var table = _console?.table ?? log;
var error = _console?.error ?? log;
var warn = _console?.warn ?? error;
var createTask = _console?.createTask ?? /* @__PURE__ */ notImplemented("console.createTask");
var clear = _console?.clear ?? noop_default;
var count = _console?.count ?? noop_default;
var countReset = _console?.countReset ?? noop_default;
var dir = _console?.dir ?? noop_default;
var dirxml = _console?.dirxml ?? noop_default;
var group = _console?.group ?? noop_default;
var groupEnd = _console?.groupEnd ?? noop_default;
var groupCollapsed = _console?.groupCollapsed ?? noop_default;
var profile = _console?.profile ?? noop_default;
var profileEnd = _console?.profileEnd ?? noop_default;
var time = _console?.time ?? noop_default;
var timeEnd = _console?.timeEnd ?? noop_default;
var timeLog = _console?.timeLog ?? noop_default;
var timeStamp = _console?.timeStamp ?? noop_default;
var Console = _console?.Console ?? /* @__PURE__ */ notImplementedClass("console.Console");
var _times = /* @__PURE__ */ new Map();
var _stdoutErrorHandler = noop_default;
var _stderrErrorHandler = noop_default;

// node_modules/@cloudflare/unenv-preset/dist/runtime/node/console.mjs
var workerdConsole = globalThis["console"];
var {
  assert,
  clear: clear2,
  // @ts-expect-error undocumented public API
  context,
  count: count2,
  countReset: countReset2,
  // @ts-expect-error undocumented public API
  createTask: createTask2,
  debug: debug2,
  dir: dir2,
  dirxml: dirxml2,
  error: error2,
  group: group2,
  groupCollapsed: groupCollapsed2,
  groupEnd: groupEnd2,
  info: info2,
  log: log2,
  profile: profile2,
  profileEnd: profileEnd2,
  table: table2,
  time: time2,
  timeEnd: timeEnd2,
  timeLog: timeLog2,
  timeStamp: timeStamp2,
  trace: trace2,
  warn: warn2
} = workerdConsole;
Object.assign(workerdConsole, {
  Console,
  _ignoreErrors,
  _stderr,
  _stderrErrorHandler,
  _stdout,
  _stdoutErrorHandler,
  _times
});
var console_default = workerdConsole;

// node_modules/wrangler/_virtual_unenv_global_polyfill-@cloudflare-unenv-preset-node-console
globalThis.console = console_default;

// node_modules/unenv/dist/runtime/node/internal/process/hrtime.mjs
var hrtime = /* @__PURE__ */ Object.assign(/* @__PURE__ */ __name(function hrtime2(startTime) {
  const now = Date.now();
  const seconds = Math.trunc(now / 1e3);
  const nanos = now % 1e3 * 1e6;
  if (startTime) {
    let diffSeconds = seconds - startTime[0];
    let diffNanos = nanos - startTime[0];
    if (diffNanos < 0) {
      diffSeconds = diffSeconds - 1;
      diffNanos = 1e9 + diffNanos;
    }
    return [diffSeconds, diffNanos];
  }
  return [seconds, nanos];
}, "hrtime"), { bigint: /* @__PURE__ */ __name(function bigint() {
  return BigInt(Date.now() * 1e6);
}, "bigint") });

// node_modules/unenv/dist/runtime/node/internal/process/process.mjs
import { EventEmitter } from "node:events";

// node_modules/unenv/dist/runtime/node/internal/tty/read-stream.mjs
var ReadStream = class {
  static {
    __name(this, "ReadStream");
  }
  fd;
  isRaw = false;
  isTTY = false;
  constructor(fd) {
    this.fd = fd;
  }
  setRawMode(mode) {
    this.isRaw = mode;
    return this;
  }
};

// node_modules/unenv/dist/runtime/node/internal/tty/write-stream.mjs
var WriteStream = class {
  static {
    __name(this, "WriteStream");
  }
  fd;
  columns = 80;
  rows = 24;
  isTTY = false;
  constructor(fd) {
    this.fd = fd;
  }
  clearLine(dir3, callback) {
    callback && callback();
    return false;
  }
  clearScreenDown(callback) {
    callback && callback();
    return false;
  }
  cursorTo(x, y, callback) {
    callback && typeof callback === "function" && callback();
    return false;
  }
  moveCursor(dx, dy, callback) {
    callback && callback();
    return false;
  }
  getColorDepth(env2) {
    return 1;
  }
  hasColors(count3, env2) {
    return false;
  }
  getWindowSize() {
    return [this.columns, this.rows];
  }
  write(str, encoding, cb) {
    if (str instanceof Uint8Array) {
      str = new TextDecoder().decode(str);
    }
    try {
      console.log(str);
    } catch {
    }
    cb && typeof cb === "function" && cb();
    return false;
  }
};

// node_modules/unenv/dist/runtime/node/internal/process/node-version.mjs
var NODE_VERSION = "22.14.0";

// node_modules/unenv/dist/runtime/node/internal/process/process.mjs
var Process = class _Process extends EventEmitter {
  static {
    __name(this, "Process");
  }
  env;
  hrtime;
  nextTick;
  constructor(impl) {
    super();
    this.env = impl.env;
    this.hrtime = impl.hrtime;
    this.nextTick = impl.nextTick;
    for (const prop of [...Object.getOwnPropertyNames(_Process.prototype), ...Object.getOwnPropertyNames(EventEmitter.prototype)]) {
      const value = this[prop];
      if (typeof value === "function") {
        this[prop] = value.bind(this);
      }
    }
  }
  // --- event emitter ---
  emitWarning(warning, type, code) {
    console.warn(`${code ? `[${code}] ` : ""}${type ? `${type}: ` : ""}${warning}`);
  }
  emit(...args) {
    return super.emit(...args);
  }
  listeners(eventName) {
    return super.listeners(eventName);
  }
  // --- stdio (lazy initializers) ---
  #stdin;
  #stdout;
  #stderr;
  get stdin() {
    return this.#stdin ??= new ReadStream(0);
  }
  get stdout() {
    return this.#stdout ??= new WriteStream(1);
  }
  get stderr() {
    return this.#stderr ??= new WriteStream(2);
  }
  // --- cwd ---
  #cwd = "/";
  chdir(cwd2) {
    this.#cwd = cwd2;
  }
  cwd() {
    return this.#cwd;
  }
  // --- dummy props and getters ---
  arch = "";
  platform = "";
  argv = [];
  argv0 = "";
  execArgv = [];
  execPath = "";
  title = "";
  pid = 200;
  ppid = 100;
  get version() {
    return `v${NODE_VERSION}`;
  }
  get versions() {
    return { node: NODE_VERSION };
  }
  get allowedNodeEnvironmentFlags() {
    return /* @__PURE__ */ new Set();
  }
  get sourceMapsEnabled() {
    return false;
  }
  get debugPort() {
    return 0;
  }
  get throwDeprecation() {
    return false;
  }
  get traceDeprecation() {
    return false;
  }
  get features() {
    return {};
  }
  get release() {
    return {};
  }
  get connected() {
    return false;
  }
  get config() {
    return {};
  }
  get moduleLoadList() {
    return [];
  }
  constrainedMemory() {
    return 0;
  }
  availableMemory() {
    return 0;
  }
  uptime() {
    return 0;
  }
  resourceUsage() {
    return {};
  }
  // --- noop methods ---
  ref() {
  }
  unref() {
  }
  // --- unimplemented methods ---
  umask() {
    throw createNotImplementedError("process.umask");
  }
  getBuiltinModule() {
    return void 0;
  }
  getActiveResourcesInfo() {
    throw createNotImplementedError("process.getActiveResourcesInfo");
  }
  exit() {
    throw createNotImplementedError("process.exit");
  }
  reallyExit() {
    throw createNotImplementedError("process.reallyExit");
  }
  kill() {
    throw createNotImplementedError("process.kill");
  }
  abort() {
    throw createNotImplementedError("process.abort");
  }
  dlopen() {
    throw createNotImplementedError("process.dlopen");
  }
  setSourceMapsEnabled() {
    throw createNotImplementedError("process.setSourceMapsEnabled");
  }
  loadEnvFile() {
    throw createNotImplementedError("process.loadEnvFile");
  }
  disconnect() {
    throw createNotImplementedError("process.disconnect");
  }
  cpuUsage() {
    throw createNotImplementedError("process.cpuUsage");
  }
  setUncaughtExceptionCaptureCallback() {
    throw createNotImplementedError("process.setUncaughtExceptionCaptureCallback");
  }
  hasUncaughtExceptionCaptureCallback() {
    throw createNotImplementedError("process.hasUncaughtExceptionCaptureCallback");
  }
  initgroups() {
    throw createNotImplementedError("process.initgroups");
  }
  openStdin() {
    throw createNotImplementedError("process.openStdin");
  }
  assert() {
    throw createNotImplementedError("process.assert");
  }
  binding() {
    throw createNotImplementedError("process.binding");
  }
  // --- attached interfaces ---
  permission = { has: /* @__PURE__ */ notImplemented("process.permission.has") };
  report = {
    directory: "",
    filename: "",
    signal: "SIGUSR2",
    compact: false,
    reportOnFatalError: false,
    reportOnSignal: false,
    reportOnUncaughtException: false,
    getReport: /* @__PURE__ */ notImplemented("process.report.getReport"),
    writeReport: /* @__PURE__ */ notImplemented("process.report.writeReport")
  };
  finalization = {
    register: /* @__PURE__ */ notImplemented("process.finalization.register"),
    unregister: /* @__PURE__ */ notImplemented("process.finalization.unregister"),
    registerBeforeExit: /* @__PURE__ */ notImplemented("process.finalization.registerBeforeExit")
  };
  memoryUsage = Object.assign(() => ({
    arrayBuffers: 0,
    rss: 0,
    external: 0,
    heapTotal: 0,
    heapUsed: 0
  }), { rss: /* @__PURE__ */ __name(() => 0, "rss") });
  // --- undefined props ---
  mainModule = void 0;
  domain = void 0;
  // optional
  send = void 0;
  exitCode = void 0;
  channel = void 0;
  getegid = void 0;
  geteuid = void 0;
  getgid = void 0;
  getgroups = void 0;
  getuid = void 0;
  setegid = void 0;
  seteuid = void 0;
  setgid = void 0;
  setgroups = void 0;
  setuid = void 0;
  // internals
  _events = void 0;
  _eventsCount = void 0;
  _exiting = void 0;
  _maxListeners = void 0;
  _debugEnd = void 0;
  _debugProcess = void 0;
  _fatalException = void 0;
  _getActiveHandles = void 0;
  _getActiveRequests = void 0;
  _kill = void 0;
  _preload_modules = void 0;
  _rawDebug = void 0;
  _startProfilerIdleNotifier = void 0;
  _stopProfilerIdleNotifier = void 0;
  _tickCallback = void 0;
  _disconnect = void 0;
  _handleQueue = void 0;
  _pendingMessage = void 0;
  _channel = void 0;
  _send = void 0;
  _linkedBinding = void 0;
};

// node_modules/@cloudflare/unenv-preset/dist/runtime/node/process.mjs
var globalProcess = globalThis["process"];
var getBuiltinModule = globalProcess.getBuiltinModule;
var workerdProcess = getBuiltinModule("node:process");
var isWorkerdProcessV2 = globalThis.Cloudflare.compatibilityFlags.enable_nodejs_process_v2;
var unenvProcess = new Process({
  env: globalProcess.env,
  // `hrtime` is only available from workerd process v2
  hrtime: isWorkerdProcessV2 ? workerdProcess.hrtime : hrtime,
  // `nextTick` is available from workerd process v1
  nextTick: workerdProcess.nextTick
});
var { exit, features, platform } = workerdProcess;
var {
  // Always implemented by workerd
  env,
  // Only implemented in workerd v2
  hrtime: hrtime3,
  // Always implemented by workerd
  nextTick
} = unenvProcess;
var {
  _channel,
  _disconnect,
  _events,
  _eventsCount,
  _handleQueue,
  _maxListeners,
  _pendingMessage,
  _send,
  assert: assert2,
  disconnect,
  mainModule
} = unenvProcess;
var {
  // @ts-expect-error `_debugEnd` is missing typings
  _debugEnd,
  // @ts-expect-error `_debugProcess` is missing typings
  _debugProcess,
  // @ts-expect-error `_exiting` is missing typings
  _exiting,
  // @ts-expect-error `_fatalException` is missing typings
  _fatalException,
  // @ts-expect-error `_getActiveHandles` is missing typings
  _getActiveHandles,
  // @ts-expect-error `_getActiveRequests` is missing typings
  _getActiveRequests,
  // @ts-expect-error `_kill` is missing typings
  _kill,
  // @ts-expect-error `_linkedBinding` is missing typings
  _linkedBinding,
  // @ts-expect-error `_preload_modules` is missing typings
  _preload_modules,
  // @ts-expect-error `_rawDebug` is missing typings
  _rawDebug,
  // @ts-expect-error `_startProfilerIdleNotifier` is missing typings
  _startProfilerIdleNotifier,
  // @ts-expect-error `_stopProfilerIdleNotifier` is missing typings
  _stopProfilerIdleNotifier,
  // @ts-expect-error `_tickCallback` is missing typings
  _tickCallback,
  abort,
  addListener,
  allowedNodeEnvironmentFlags,
  arch,
  argv,
  argv0,
  availableMemory,
  // @ts-expect-error `binding` is missing typings
  binding,
  channel,
  chdir,
  config,
  connected,
  constrainedMemory,
  cpuUsage,
  cwd,
  debugPort,
  dlopen,
  // @ts-expect-error `domain` is missing typings
  domain,
  emit,
  emitWarning,
  eventNames,
  execArgv,
  execPath,
  exitCode,
  finalization,
  getActiveResourcesInfo,
  getegid,
  geteuid,
  getgid,
  getgroups,
  getMaxListeners,
  getuid,
  hasUncaughtExceptionCaptureCallback,
  // @ts-expect-error `initgroups` is missing typings
  initgroups,
  kill,
  listenerCount,
  listeners,
  loadEnvFile,
  memoryUsage,
  // @ts-expect-error `moduleLoadList` is missing typings
  moduleLoadList,
  off,
  on,
  once,
  // @ts-expect-error `openStdin` is missing typings
  openStdin,
  permission,
  pid,
  ppid,
  prependListener,
  prependOnceListener,
  rawListeners,
  // @ts-expect-error `reallyExit` is missing typings
  reallyExit,
  ref,
  release,
  removeAllListeners,
  removeListener,
  report,
  resourceUsage,
  send,
  setegid,
  seteuid,
  setgid,
  setgroups,
  setMaxListeners,
  setSourceMapsEnabled,
  setuid,
  setUncaughtExceptionCaptureCallback,
  sourceMapsEnabled,
  stderr,
  stdin,
  stdout,
  throwDeprecation,
  title,
  traceDeprecation,
  umask,
  unref,
  uptime,
  version,
  versions
} = isWorkerdProcessV2 ? workerdProcess : unenvProcess;
var _process = {
  abort,
  addListener,
  allowedNodeEnvironmentFlags,
  hasUncaughtExceptionCaptureCallback,
  setUncaughtExceptionCaptureCallback,
  loadEnvFile,
  sourceMapsEnabled,
  arch,
  argv,
  argv0,
  chdir,
  config,
  connected,
  constrainedMemory,
  availableMemory,
  cpuUsage,
  cwd,
  debugPort,
  dlopen,
  disconnect,
  emit,
  emitWarning,
  env,
  eventNames,
  execArgv,
  execPath,
  exit,
  finalization,
  features,
  getBuiltinModule,
  getActiveResourcesInfo,
  getMaxListeners,
  hrtime: hrtime3,
  kill,
  listeners,
  listenerCount,
  memoryUsage,
  nextTick,
  on,
  off,
  once,
  pid,
  platform,
  ppid,
  prependListener,
  prependOnceListener,
  rawListeners,
  release,
  removeAllListeners,
  removeListener,
  report,
  resourceUsage,
  setMaxListeners,
  setSourceMapsEnabled,
  stderr,
  stdin,
  stdout,
  title,
  throwDeprecation,
  traceDeprecation,
  umask,
  uptime,
  version,
  versions,
  // @ts-expect-error old API
  domain,
  initgroups,
  moduleLoadList,
  reallyExit,
  openStdin,
  assert: assert2,
  binding,
  send,
  exitCode,
  channel,
  getegid,
  geteuid,
  getgid,
  getgroups,
  getuid,
  setegid,
  seteuid,
  setgid,
  setgroups,
  setuid,
  permission,
  mainModule,
  _events,
  _eventsCount,
  _exiting,
  _maxListeners,
  _debugEnd,
  _debugProcess,
  _fatalException,
  _getActiveHandles,
  _getActiveRequests,
  _kill,
  _preload_modules,
  _rawDebug,
  _startProfilerIdleNotifier,
  _stopProfilerIdleNotifier,
  _tickCallback,
  _disconnect,
  _handleQueue,
  _pendingMessage,
  _channel,
  _send,
  _linkedBinding
};
var process_default = _process;

// node_modules/wrangler/_virtual_unenv_global_polyfill-@cloudflare-unenv-preset-node-process
globalThis.process = process_default;

// node_modules/hono/dist/compose.js
var compose = /* @__PURE__ */ __name((middleware, onError, onNotFound) => {
  return (context2, next) => {
    let index = -1;
    return dispatch(0);
    async function dispatch(i3) {
      if (i3 <= index) {
        throw new Error("next() called multiple times");
      }
      index = i3;
      let res;
      let isError = false;
      let handler;
      if (middleware[i3]) {
        handler = middleware[i3][0][0];
        context2.req.routeIndex = i3;
      } else {
        handler = i3 === middleware.length && next || void 0;
      }
      if (handler) {
        try {
          res = await handler(context2, () => dispatch(i3 + 1));
        } catch (err) {
          if (err instanceof Error && onError) {
            context2.error = err;
            res = await onError(err, context2);
            isError = true;
          } else {
            throw err;
          }
        }
      } else {
        if (context2.finalized === false && onNotFound) {
          res = await onNotFound(context2);
        }
      }
      if (res && (context2.finalized === false || isError)) {
        context2.res = res;
      }
      return context2;
    }
    __name(dispatch, "dispatch");
  };
}, "compose");

// node_modules/hono/dist/request/constants.js
var GET_MATCH_RESULT = /* @__PURE__ */ Symbol();

// node_modules/hono/dist/utils/body.js
var parseBody = /* @__PURE__ */ __name(async (request, options = /* @__PURE__ */ Object.create(null)) => {
  const { all = false, dot = false } = options;
  const headers = request instanceof HonoRequest ? request.raw.headers : request.headers;
  const contentType = headers.get("Content-Type");
  if (contentType?.startsWith("multipart/form-data") || contentType?.startsWith("application/x-www-form-urlencoded")) {
    return parseFormData(request, { all, dot });
  }
  return {};
}, "parseBody");
async function parseFormData(request, options) {
  const formData = await request.formData();
  if (formData) {
    return convertFormDataToBodyData(formData, options);
  }
  return {};
}
__name(parseFormData, "parseFormData");
function convertFormDataToBodyData(formData, options) {
  const form = /* @__PURE__ */ Object.create(null);
  formData.forEach((value, key) => {
    const shouldParseAllValues = options.all || key.endsWith("[]");
    if (!shouldParseAllValues) {
      form[key] = value;
    } else {
      handleParsingAllValues(form, key, value);
    }
  });
  if (options.dot) {
    Object.entries(form).forEach(([key, value]) => {
      const shouldParseDotValues = key.includes(".");
      if (shouldParseDotValues) {
        handleParsingNestedValues(form, key, value);
        delete form[key];
      }
    });
  }
  return form;
}
__name(convertFormDataToBodyData, "convertFormDataToBodyData");
var handleParsingAllValues = /* @__PURE__ */ __name((form, key, value) => {
  if (form[key] !== void 0) {
    if (Array.isArray(form[key])) {
      ;
      form[key].push(value);
    } else {
      form[key] = [form[key], value];
    }
  } else {
    if (!key.endsWith("[]")) {
      form[key] = value;
    } else {
      form[key] = [value];
    }
  }
}, "handleParsingAllValues");
var handleParsingNestedValues = /* @__PURE__ */ __name((form, key, value) => {
  let nestedForm = form;
  const keys = key.split(".");
  keys.forEach((key2, index) => {
    if (index === keys.length - 1) {
      nestedForm[key2] = value;
    } else {
      if (!nestedForm[key2] || typeof nestedForm[key2] !== "object" || Array.isArray(nestedForm[key2]) || nestedForm[key2] instanceof File) {
        nestedForm[key2] = /* @__PURE__ */ Object.create(null);
      }
      nestedForm = nestedForm[key2];
    }
  });
}, "handleParsingNestedValues");

// node_modules/hono/dist/utils/url.js
var splitPath = /* @__PURE__ */ __name((path) => {
  const paths = path.split("/");
  if (paths[0] === "") {
    paths.shift();
  }
  return paths;
}, "splitPath");
var splitRoutingPath = /* @__PURE__ */ __name((routePath) => {
  const { groups, path } = extractGroupsFromPath(routePath);
  const paths = splitPath(path);
  return replaceGroupMarks(paths, groups);
}, "splitRoutingPath");
var extractGroupsFromPath = /* @__PURE__ */ __name((path) => {
  const groups = [];
  path = path.replace(/\{[^}]+\}/g, (match2, index) => {
    const mark = `@${index}`;
    groups.push([mark, match2]);
    return mark;
  });
  return { groups, path };
}, "extractGroupsFromPath");
var replaceGroupMarks = /* @__PURE__ */ __name((paths, groups) => {
  for (let i3 = groups.length - 1; i3 >= 0; i3--) {
    const [mark] = groups[i3];
    for (let j = paths.length - 1; j >= 0; j--) {
      if (paths[j].includes(mark)) {
        paths[j] = paths[j].replace(mark, groups[i3][1]);
        break;
      }
    }
  }
  return paths;
}, "replaceGroupMarks");
var patternCache = {};
var getPattern = /* @__PURE__ */ __name((label, next) => {
  if (label === "*") {
    return "*";
  }
  const match2 = label.match(/^\:([^\{\}]+)(?:\{(.+)\})?$/);
  if (match2) {
    const cacheKey = `${label}#${next}`;
    if (!patternCache[cacheKey]) {
      if (match2[2]) {
        patternCache[cacheKey] = next && next[0] !== ":" && next[0] !== "*" ? [cacheKey, match2[1], new RegExp(`^${match2[2]}(?=/${next})`)] : [label, match2[1], new RegExp(`^${match2[2]}$`)];
      } else {
        patternCache[cacheKey] = [label, match2[1], true];
      }
    }
    return patternCache[cacheKey];
  }
  return null;
}, "getPattern");
var tryDecode = /* @__PURE__ */ __name((str, decoder) => {
  try {
    return decoder(str);
  } catch {
    return str.replace(/(?:%[0-9A-Fa-f]{2})+/g, (match2) => {
      try {
        return decoder(match2);
      } catch {
        return match2;
      }
    });
  }
}, "tryDecode");
var tryDecodeURI = /* @__PURE__ */ __name((str) => tryDecode(str, decodeURI), "tryDecodeURI");
var getPath = /* @__PURE__ */ __name((request) => {
  const url = request.url;
  const start = url.indexOf("/", url.indexOf(":") + 4);
  let i3 = start;
  for (; i3 < url.length; i3++) {
    const charCode = url.charCodeAt(i3);
    if (charCode === 37) {
      const queryIndex = url.indexOf("?", i3);
      const path = url.slice(start, queryIndex === -1 ? void 0 : queryIndex);
      return tryDecodeURI(path.includes("%25") ? path.replace(/%25/g, "%2525") : path);
    } else if (charCode === 63) {
      break;
    }
  }
  return url.slice(start, i3);
}, "getPath");
var getPathNoStrict = /* @__PURE__ */ __name((request) => {
  const result = getPath(request);
  return result.length > 1 && result.at(-1) === "/" ? result.slice(0, -1) : result;
}, "getPathNoStrict");
var mergePath = /* @__PURE__ */ __name((base, sub, ...rest) => {
  if (rest.length) {
    sub = mergePath(sub, ...rest);
  }
  return `${base?.[0] === "/" ? "" : "/"}${base}${sub === "/" ? "" : `${base?.at(-1) === "/" ? "" : "/"}${sub?.[0] === "/" ? sub.slice(1) : sub}`}`;
}, "mergePath");
var checkOptionalParameter = /* @__PURE__ */ __name((path) => {
  if (path.charCodeAt(path.length - 1) !== 63 || !path.includes(":")) {
    return null;
  }
  const segments = path.split("/");
  const results = [];
  let basePath = "";
  segments.forEach((segment) => {
    if (segment !== "" && !/\:/.test(segment)) {
      basePath += "/" + segment;
    } else if (/\:/.test(segment)) {
      if (/\?/.test(segment)) {
        if (results.length === 0 && basePath === "") {
          results.push("/");
        } else {
          results.push(basePath);
        }
        const optionalSegment = segment.replace("?", "");
        basePath += "/" + optionalSegment;
        results.push(basePath);
      } else {
        basePath += "/" + segment;
      }
    }
  });
  return results.filter((v, i3, a3) => a3.indexOf(v) === i3);
}, "checkOptionalParameter");
var _decodeURI = /* @__PURE__ */ __name((value) => {
  if (!/[%+]/.test(value)) {
    return value;
  }
  if (value.indexOf("+") !== -1) {
    value = value.replace(/\+/g, " ");
  }
  return value.indexOf("%") !== -1 ? tryDecode(value, decodeURIComponent_) : value;
}, "_decodeURI");
var _getQueryParam = /* @__PURE__ */ __name((url, key, multiple) => {
  let encoded;
  if (!multiple && key && !/[%+]/.test(key)) {
    let keyIndex2 = url.indexOf("?", 8);
    if (keyIndex2 === -1) {
      return void 0;
    }
    if (!url.startsWith(key, keyIndex2 + 1)) {
      keyIndex2 = url.indexOf(`&${key}`, keyIndex2 + 1);
    }
    while (keyIndex2 !== -1) {
      const trailingKeyCode = url.charCodeAt(keyIndex2 + key.length + 1);
      if (trailingKeyCode === 61) {
        const valueIndex = keyIndex2 + key.length + 2;
        const endIndex = url.indexOf("&", valueIndex);
        return _decodeURI(url.slice(valueIndex, endIndex === -1 ? void 0 : endIndex));
      } else if (trailingKeyCode == 38 || isNaN(trailingKeyCode)) {
        return "";
      }
      keyIndex2 = url.indexOf(`&${key}`, keyIndex2 + 1);
    }
    encoded = /[%+]/.test(url);
    if (!encoded) {
      return void 0;
    }
  }
  const results = {};
  encoded ??= /[%+]/.test(url);
  let keyIndex = url.indexOf("?", 8);
  while (keyIndex !== -1) {
    const nextKeyIndex = url.indexOf("&", keyIndex + 1);
    let valueIndex = url.indexOf("=", keyIndex);
    if (valueIndex > nextKeyIndex && nextKeyIndex !== -1) {
      valueIndex = -1;
    }
    let name = url.slice(
      keyIndex + 1,
      valueIndex === -1 ? nextKeyIndex === -1 ? void 0 : nextKeyIndex : valueIndex
    );
    if (encoded) {
      name = _decodeURI(name);
    }
    keyIndex = nextKeyIndex;
    if (name === "") {
      continue;
    }
    let value;
    if (valueIndex === -1) {
      value = "";
    } else {
      value = url.slice(valueIndex + 1, nextKeyIndex === -1 ? void 0 : nextKeyIndex);
      if (encoded) {
        value = _decodeURI(value);
      }
    }
    if (multiple) {
      if (!(results[name] && Array.isArray(results[name]))) {
        results[name] = [];
      }
      ;
      results[name].push(value);
    } else {
      results[name] ??= value;
    }
  }
  return key ? results[key] : results;
}, "_getQueryParam");
var getQueryParam = _getQueryParam;
var getQueryParams = /* @__PURE__ */ __name((url, key) => {
  return _getQueryParam(url, key, true);
}, "getQueryParams");
var decodeURIComponent_ = decodeURIComponent;

// node_modules/hono/dist/request.js
var tryDecodeURIComponent = /* @__PURE__ */ __name((str) => tryDecode(str, decodeURIComponent_), "tryDecodeURIComponent");
var HonoRequest = class {
  static {
    __name(this, "HonoRequest");
  }
  /**
   * `.raw` can get the raw Request object.
   *
   * @see {@link https://hono.dev/docs/api/request#raw}
   *
   * @example
   * ```ts
   * // For Cloudflare Workers
   * app.post('/', async (c) => {
   *   const metadata = c.req.raw.cf?.hostMetadata?
   *   ...
   * })
   * ```
   */
  raw;
  #validatedData;
  // Short name of validatedData
  #matchResult;
  routeIndex = 0;
  /**
   * `.path` can get the pathname of the request.
   *
   * @see {@link https://hono.dev/docs/api/request#path}
   *
   * @example
   * ```ts
   * app.get('/about/me', (c) => {
   *   const pathname = c.req.path // `/about/me`
   * })
   * ```
   */
  path;
  bodyCache = {};
  constructor(request, path = "/", matchResult = [[]]) {
    this.raw = request;
    this.path = path;
    this.#matchResult = matchResult;
    this.#validatedData = {};
  }
  param(key) {
    return key ? this.#getDecodedParam(key) : this.#getAllDecodedParams();
  }
  #getDecodedParam(key) {
    const paramKey = this.#matchResult[0][this.routeIndex][1][key];
    const param = this.#getParamValue(paramKey);
    return param && /\%/.test(param) ? tryDecodeURIComponent(param) : param;
  }
  #getAllDecodedParams() {
    const decoded = {};
    const keys = Object.keys(this.#matchResult[0][this.routeIndex][1]);
    for (const key of keys) {
      const value = this.#getParamValue(this.#matchResult[0][this.routeIndex][1][key]);
      if (value !== void 0) {
        decoded[key] = /\%/.test(value) ? tryDecodeURIComponent(value) : value;
      }
    }
    return decoded;
  }
  #getParamValue(paramKey) {
    return this.#matchResult[1] ? this.#matchResult[1][paramKey] : paramKey;
  }
  query(key) {
    return getQueryParam(this.url, key);
  }
  queries(key) {
    return getQueryParams(this.url, key);
  }
  header(name) {
    if (name) {
      return this.raw.headers.get(name) ?? void 0;
    }
    const headerData = {};
    this.raw.headers.forEach((value, key) => {
      headerData[key] = value;
    });
    return headerData;
  }
  async parseBody(options) {
    return this.bodyCache.parsedBody ??= await parseBody(this, options);
  }
  #cachedBody = /* @__PURE__ */ __name((key) => {
    const { bodyCache, raw: raw2 } = this;
    const cachedBody = bodyCache[key];
    if (cachedBody) {
      return cachedBody;
    }
    const anyCachedKey = Object.keys(bodyCache)[0];
    if (anyCachedKey) {
      return bodyCache[anyCachedKey].then((body) => {
        if (anyCachedKey === "json") {
          body = JSON.stringify(body);
        }
        return new Response(body)[key]();
      });
    }
    return bodyCache[key] = raw2[key]();
  }, "#cachedBody");
  /**
   * `.json()` can parse Request body of type `application/json`
   *
   * @see {@link https://hono.dev/docs/api/request#json}
   *
   * @example
   * ```ts
   * app.post('/entry', async (c) => {
   *   const body = await c.req.json()
   * })
   * ```
   */
  json() {
    return this.#cachedBody("text").then((text) => JSON.parse(text));
  }
  /**
   * `.text()` can parse Request body of type `text/plain`
   *
   * @see {@link https://hono.dev/docs/api/request#text}
   *
   * @example
   * ```ts
   * app.post('/entry', async (c) => {
   *   const body = await c.req.text()
   * })
   * ```
   */
  text() {
    return this.#cachedBody("text");
  }
  /**
   * `.arrayBuffer()` parse Request body as an `ArrayBuffer`
   *
   * @see {@link https://hono.dev/docs/api/request#arraybuffer}
   *
   * @example
   * ```ts
   * app.post('/entry', async (c) => {
   *   const body = await c.req.arrayBuffer()
   * })
   * ```
   */
  arrayBuffer() {
    return this.#cachedBody("arrayBuffer");
  }
  /**
   * Parses the request body as a `Blob`.
   * @example
   * ```ts
   * app.post('/entry', async (c) => {
   *   const body = await c.req.blob();
   * });
   * ```
   * @see https://hono.dev/docs/api/request#blob
   */
  blob() {
    return this.#cachedBody("blob");
  }
  /**
   * Parses the request body as `FormData`.
   * @example
   * ```ts
   * app.post('/entry', async (c) => {
   *   const body = await c.req.formData();
   * });
   * ```
   * @see https://hono.dev/docs/api/request#formdata
   */
  formData() {
    return this.#cachedBody("formData");
  }
  /**
   * Adds validated data to the request.
   *
   * @param target - The target of the validation.
   * @param data - The validated data to add.
   */
  addValidatedData(target, data) {
    this.#validatedData[target] = data;
  }
  valid(target) {
    return this.#validatedData[target];
  }
  /**
   * `.url()` can get the request url strings.
   *
   * @see {@link https://hono.dev/docs/api/request#url}
   *
   * @example
   * ```ts
   * app.get('/about/me', (c) => {
   *   const url = c.req.url // `http://localhost:8787/about/me`
   *   ...
   * })
   * ```
   */
  get url() {
    return this.raw.url;
  }
  /**
   * `.method()` can get the method name of the request.
   *
   * @see {@link https://hono.dev/docs/api/request#method}
   *
   * @example
   * ```ts
   * app.get('/about/me', (c) => {
   *   const method = c.req.method // `GET`
   * })
   * ```
   */
  get method() {
    return this.raw.method;
  }
  get [GET_MATCH_RESULT]() {
    return this.#matchResult;
  }
  /**
   * `.matchedRoutes()` can return a matched route in the handler
   *
   * @deprecated
   *
   * Use matchedRoutes helper defined in "hono/route" instead.
   *
   * @see {@link https://hono.dev/docs/api/request#matchedroutes}
   *
   * @example
   * ```ts
   * app.use('*', async function logger(c, next) {
   *   await next()
   *   c.req.matchedRoutes.forEach(({ handler, method, path }, i) => {
   *     const name = handler.name || (handler.length < 2 ? '[handler]' : '[middleware]')
   *     console.log(
   *       method,
   *       ' ',
   *       path,
   *       ' '.repeat(Math.max(10 - path.length, 0)),
   *       name,
   *       i === c.req.routeIndex ? '<- respond from here' : ''
   *     )
   *   })
   * })
   * ```
   */
  get matchedRoutes() {
    return this.#matchResult[0].map(([[, route]]) => route);
  }
  /**
   * `routePath()` can retrieve the path registered within the handler
   *
   * @deprecated
   *
   * Use routePath helper defined in "hono/route" instead.
   *
   * @see {@link https://hono.dev/docs/api/request#routepath}
   *
   * @example
   * ```ts
   * app.get('/posts/:id', (c) => {
   *   return c.json({ path: c.req.routePath })
   * })
   * ```
   */
  get routePath() {
    return this.#matchResult[0].map(([[, route]]) => route)[this.routeIndex].path;
  }
};

// node_modules/hono/dist/utils/html.js
var HtmlEscapedCallbackPhase = {
  Stringify: 1,
  BeforeStream: 2,
  Stream: 3
};
var raw = /* @__PURE__ */ __name((value, callbacks) => {
  const escapedString = new String(value);
  escapedString.isEscaped = true;
  escapedString.callbacks = callbacks;
  return escapedString;
}, "raw");
var resolveCallback = /* @__PURE__ */ __name(async (str, phase, preserveCallbacks, context2, buffer) => {
  if (typeof str === "object" && !(str instanceof String)) {
    if (!(str instanceof Promise)) {
      str = str.toString();
    }
    if (str instanceof Promise) {
      str = await str;
    }
  }
  const callbacks = str.callbacks;
  if (!callbacks?.length) {
    return Promise.resolve(str);
  }
  if (buffer) {
    buffer[0] += str;
  } else {
    buffer = [str];
  }
  const resStr = Promise.all(callbacks.map((c) => c({ phase, buffer, context: context2 }))).then(
    (res) => Promise.all(
      res.filter(Boolean).map((str2) => resolveCallback(str2, phase, false, context2, buffer))
    ).then(() => buffer[0])
  );
  if (preserveCallbacks) {
    return raw(await resStr, callbacks);
  } else {
    return resStr;
  }
}, "resolveCallback");

// node_modules/hono/dist/context.js
var TEXT_PLAIN = "text/plain; charset=UTF-8";
var setDefaultContentType = /* @__PURE__ */ __name((contentType, headers) => {
  return {
    "Content-Type": contentType,
    ...headers
  };
}, "setDefaultContentType");
var Context = class {
  static {
    __name(this, "Context");
  }
  #rawRequest;
  #req;
  /**
   * `.env` can get bindings (environment variables, secrets, KV namespaces, D1 database, R2 bucket etc.) in Cloudflare Workers.
   *
   * @see {@link https://hono.dev/docs/api/context#env}
   *
   * @example
   * ```ts
   * // Environment object for Cloudflare Workers
   * app.get('*', async c => {
   *   const counter = c.env.COUNTER
   * })
   * ```
   */
  env = {};
  #var;
  finalized = false;
  /**
   * `.error` can get the error object from the middleware if the Handler throws an error.
   *
   * @see {@link https://hono.dev/docs/api/context#error}
   *
   * @example
   * ```ts
   * app.use('*', async (c, next) => {
   *   await next()
   *   if (c.error) {
   *     // do something...
   *   }
   * })
   * ```
   */
  error;
  #status;
  #executionCtx;
  #res;
  #layout;
  #renderer;
  #notFoundHandler;
  #preparedHeaders;
  #matchResult;
  #path;
  /**
   * Creates an instance of the Context class.
   *
   * @param req - The Request object.
   * @param options - Optional configuration options for the context.
   */
  constructor(req, options) {
    this.#rawRequest = req;
    if (options) {
      this.#executionCtx = options.executionCtx;
      this.env = options.env;
      this.#notFoundHandler = options.notFoundHandler;
      this.#path = options.path;
      this.#matchResult = options.matchResult;
    }
  }
  /**
   * `.req` is the instance of {@link HonoRequest}.
   */
  get req() {
    this.#req ??= new HonoRequest(this.#rawRequest, this.#path, this.#matchResult);
    return this.#req;
  }
  /**
   * @see {@link https://hono.dev/docs/api/context#event}
   * The FetchEvent associated with the current request.
   *
   * @throws Will throw an error if the context does not have a FetchEvent.
   */
  get event() {
    if (this.#executionCtx && "respondWith" in this.#executionCtx) {
      return this.#executionCtx;
    } else {
      throw Error("This context has no FetchEvent");
    }
  }
  /**
   * @see {@link https://hono.dev/docs/api/context#executionctx}
   * The ExecutionContext associated with the current request.
   *
   * @throws Will throw an error if the context does not have an ExecutionContext.
   */
  get executionCtx() {
    if (this.#executionCtx) {
      return this.#executionCtx;
    } else {
      throw Error("This context has no ExecutionContext");
    }
  }
  /**
   * @see {@link https://hono.dev/docs/api/context#res}
   * The Response object for the current request.
   */
  get res() {
    return this.#res ||= new Response(null, {
      headers: this.#preparedHeaders ??= new Headers()
    });
  }
  /**
   * Sets the Response object for the current request.
   *
   * @param _res - The Response object to set.
   */
  set res(_res) {
    if (this.#res && _res) {
      _res = new Response(_res.body, _res);
      for (const [k, v] of this.#res.headers.entries()) {
        if (k === "content-type") {
          continue;
        }
        if (k === "set-cookie") {
          const cookies = this.#res.headers.getSetCookie();
          _res.headers.delete("set-cookie");
          for (const cookie of cookies) {
            _res.headers.append("set-cookie", cookie);
          }
        } else {
          _res.headers.set(k, v);
        }
      }
    }
    this.#res = _res;
    this.finalized = true;
  }
  /**
   * `.render()` can create a response within a layout.
   *
   * @see {@link https://hono.dev/docs/api/context#render-setrenderer}
   *
   * @example
   * ```ts
   * app.get('/', (c) => {
   *   return c.render('Hello!')
   * })
   * ```
   */
  render = /* @__PURE__ */ __name((...args) => {
    this.#renderer ??= (content) => this.html(content);
    return this.#renderer(...args);
  }, "render");
  /**
   * Sets the layout for the response.
   *
   * @param layout - The layout to set.
   * @returns The layout function.
   */
  setLayout = /* @__PURE__ */ __name((layout) => this.#layout = layout, "setLayout");
  /**
   * Gets the current layout for the response.
   *
   * @returns The current layout function.
   */
  getLayout = /* @__PURE__ */ __name(() => this.#layout, "getLayout");
  /**
   * `.setRenderer()` can set the layout in the custom middleware.
   *
   * @see {@link https://hono.dev/docs/api/context#render-setrenderer}
   *
   * @example
   * ```tsx
   * app.use('*', async (c, next) => {
   *   c.setRenderer((content) => {
   *     return c.html(
   *       <html>
   *         <body>
   *           <p>{content}</p>
   *         </body>
   *       </html>
   *     )
   *   })
   *   await next()
   * })
   * ```
   */
  setRenderer = /* @__PURE__ */ __name((renderer) => {
    this.#renderer = renderer;
  }, "setRenderer");
  /**
   * `.header()` can set headers.
   *
   * @see {@link https://hono.dev/docs/api/context#header}
   *
   * @example
   * ```ts
   * app.get('/welcome', (c) => {
   *   // Set headers
   *   c.header('X-Message', 'Hello!')
   *   c.header('Content-Type', 'text/plain')
   *
   *   return c.body('Thank you for coming')
   * })
   * ```
   */
  header = /* @__PURE__ */ __name((name, value, options) => {
    if (this.finalized) {
      this.#res = new Response(this.#res.body, this.#res);
    }
    const headers = this.#res ? this.#res.headers : this.#preparedHeaders ??= new Headers();
    if (value === void 0) {
      headers.delete(name);
    } else if (options?.append) {
      headers.append(name, value);
    } else {
      headers.set(name, value);
    }
  }, "header");
  status = /* @__PURE__ */ __name((status) => {
    this.#status = status;
  }, "status");
  /**
   * `.set()` can set the value specified by the key.
   *
   * @see {@link https://hono.dev/docs/api/context#set-get}
   *
   * @example
   * ```ts
   * app.use('*', async (c, next) => {
   *   c.set('message', 'Hono is hot!!')
   *   await next()
   * })
   * ```
   */
  set = /* @__PURE__ */ __name((key, value) => {
    this.#var ??= /* @__PURE__ */ new Map();
    this.#var.set(key, value);
  }, "set");
  /**
   * `.get()` can use the value specified by the key.
   *
   * @see {@link https://hono.dev/docs/api/context#set-get}
   *
   * @example
   * ```ts
   * app.get('/', (c) => {
   *   const message = c.get('message')
   *   return c.text(`The message is "${message}"`)
   * })
   * ```
   */
  get = /* @__PURE__ */ __name((key) => {
    return this.#var ? this.#var.get(key) : void 0;
  }, "get");
  /**
   * `.var` can access the value of a variable.
   *
   * @see {@link https://hono.dev/docs/api/context#var}
   *
   * @example
   * ```ts
   * const result = c.var.client.oneMethod()
   * ```
   */
  // c.var.propName is a read-only
  get var() {
    if (!this.#var) {
      return {};
    }
    return Object.fromEntries(this.#var);
  }
  #newResponse(data, arg, headers) {
    const responseHeaders = this.#res ? new Headers(this.#res.headers) : this.#preparedHeaders ?? new Headers();
    if (typeof arg === "object" && "headers" in arg) {
      const argHeaders = arg.headers instanceof Headers ? arg.headers : new Headers(arg.headers);
      for (const [key, value] of argHeaders) {
        if (key.toLowerCase() === "set-cookie") {
          responseHeaders.append(key, value);
        } else {
          responseHeaders.set(key, value);
        }
      }
    }
    if (headers) {
      for (const [k, v] of Object.entries(headers)) {
        if (typeof v === "string") {
          responseHeaders.set(k, v);
        } else {
          responseHeaders.delete(k);
          for (const v2 of v) {
            responseHeaders.append(k, v2);
          }
        }
      }
    }
    const status = typeof arg === "number" ? arg : arg?.status ?? this.#status;
    return new Response(data, { status, headers: responseHeaders });
  }
  newResponse = /* @__PURE__ */ __name((...args) => this.#newResponse(...args), "newResponse");
  /**
   * `.body()` can return the HTTP response.
   * You can set headers with `.header()` and set HTTP status code with `.status`.
   * This can also be set in `.text()`, `.json()` and so on.
   *
   * @see {@link https://hono.dev/docs/api/context#body}
   *
   * @example
   * ```ts
   * app.get('/welcome', (c) => {
   *   // Set headers
   *   c.header('X-Message', 'Hello!')
   *   c.header('Content-Type', 'text/plain')
   *   // Set HTTP status code
   *   c.status(201)
   *
   *   // Return the response body
   *   return c.body('Thank you for coming')
   * })
   * ```
   */
  body = /* @__PURE__ */ __name((data, arg, headers) => this.#newResponse(data, arg, headers), "body");
  /**
   * `.text()` can render text as `Content-Type:text/plain`.
   *
   * @see {@link https://hono.dev/docs/api/context#text}
   *
   * @example
   * ```ts
   * app.get('/say', (c) => {
   *   return c.text('Hello!')
   * })
   * ```
   */
  text = /* @__PURE__ */ __name((text, arg, headers) => {
    return !this.#preparedHeaders && !this.#status && !arg && !headers && !this.finalized ? new Response(text) : this.#newResponse(
      text,
      arg,
      setDefaultContentType(TEXT_PLAIN, headers)
    );
  }, "text");
  /**
   * `.json()` can render JSON as `Content-Type:application/json`.
   *
   * @see {@link https://hono.dev/docs/api/context#json}
   *
   * @example
   * ```ts
   * app.get('/api', (c) => {
   *   return c.json({ message: 'Hello!' })
   * })
   * ```
   */
  json = /* @__PURE__ */ __name((object, arg, headers) => {
    return this.#newResponse(
      JSON.stringify(object),
      arg,
      setDefaultContentType("application/json", headers)
    );
  }, "json");
  html = /* @__PURE__ */ __name((html, arg, headers) => {
    const res = /* @__PURE__ */ __name((html2) => this.#newResponse(html2, arg, setDefaultContentType("text/html; charset=UTF-8", headers)), "res");
    return typeof html === "object" ? resolveCallback(html, HtmlEscapedCallbackPhase.Stringify, false, {}).then(res) : res(html);
  }, "html");
  /**
   * `.redirect()` can Redirect, default status code is 302.
   *
   * @see {@link https://hono.dev/docs/api/context#redirect}
   *
   * @example
   * ```ts
   * app.get('/redirect', (c) => {
   *   return c.redirect('/')
   * })
   * app.get('/redirect-permanently', (c) => {
   *   return c.redirect('/', 301)
   * })
   * ```
   */
  redirect = /* @__PURE__ */ __name((location, status) => {
    const locationString = String(location);
    this.header(
      "Location",
      // Multibyes should be encoded
      // eslint-disable-next-line no-control-regex
      !/[^\x00-\xFF]/.test(locationString) ? locationString : encodeURI(locationString)
    );
    return this.newResponse(null, status ?? 302);
  }, "redirect");
  /**
   * `.notFound()` can return the Not Found Response.
   *
   * @see {@link https://hono.dev/docs/api/context#notfound}
   *
   * @example
   * ```ts
   * app.get('/notfound', (c) => {
   *   return c.notFound()
   * })
   * ```
   */
  notFound = /* @__PURE__ */ __name(() => {
    this.#notFoundHandler ??= () => new Response();
    return this.#notFoundHandler(this);
  }, "notFound");
};

// node_modules/hono/dist/router.js
var METHOD_NAME_ALL = "ALL";
var METHOD_NAME_ALL_LOWERCASE = "all";
var METHODS = ["get", "post", "put", "delete", "options", "patch"];
var MESSAGE_MATCHER_IS_ALREADY_BUILT = "Can not add a route since the matcher is already built.";
var UnsupportedPathError = class extends Error {
  static {
    __name(this, "UnsupportedPathError");
  }
};

// node_modules/hono/dist/utils/constants.js
var COMPOSED_HANDLER = "__COMPOSED_HANDLER";

// node_modules/hono/dist/hono-base.js
var notFoundHandler = /* @__PURE__ */ __name((c) => {
  return c.text("404 Not Found", 404);
}, "notFoundHandler");
var errorHandler = /* @__PURE__ */ __name((err, c) => {
  if ("getResponse" in err) {
    const res = err.getResponse();
    return c.newResponse(res.body, res);
  }
  console.error(err);
  return c.text("Internal Server Error", 500);
}, "errorHandler");
var Hono = class _Hono {
  static {
    __name(this, "_Hono");
  }
  get;
  post;
  put;
  delete;
  options;
  patch;
  all;
  on;
  use;
  /*
    This class is like an abstract class and does not have a router.
    To use it, inherit the class and implement router in the constructor.
  */
  router;
  getPath;
  // Cannot use `#` because it requires visibility at JavaScript runtime.
  _basePath = "/";
  #path = "/";
  routes = [];
  constructor(options = {}) {
    const allMethods = [...METHODS, METHOD_NAME_ALL_LOWERCASE];
    allMethods.forEach((method) => {
      this[method] = (args1, ...args) => {
        if (typeof args1 === "string") {
          this.#path = args1;
        } else {
          this.#addRoute(method, this.#path, args1);
        }
        args.forEach((handler) => {
          this.#addRoute(method, this.#path, handler);
        });
        return this;
      };
    });
    this.on = (method, path, ...handlers) => {
      for (const p of [path].flat()) {
        this.#path = p;
        for (const m of [method].flat()) {
          handlers.map((handler) => {
            this.#addRoute(m.toUpperCase(), this.#path, handler);
          });
        }
      }
      return this;
    };
    this.use = (arg1, ...handlers) => {
      if (typeof arg1 === "string") {
        this.#path = arg1;
      } else {
        this.#path = "*";
        handlers.unshift(arg1);
      }
      handlers.forEach((handler) => {
        this.#addRoute(METHOD_NAME_ALL, this.#path, handler);
      });
      return this;
    };
    const { strict, ...optionsWithoutStrict } = options;
    Object.assign(this, optionsWithoutStrict);
    this.getPath = strict ?? true ? options.getPath ?? getPath : getPathNoStrict;
  }
  #clone() {
    const clone = new _Hono({
      router: this.router,
      getPath: this.getPath
    });
    clone.errorHandler = this.errorHandler;
    clone.#notFoundHandler = this.#notFoundHandler;
    clone.routes = this.routes;
    return clone;
  }
  #notFoundHandler = notFoundHandler;
  // Cannot use `#` because it requires visibility at JavaScript runtime.
  errorHandler = errorHandler;
  /**
   * `.route()` allows grouping other Hono instance in routes.
   *
   * @see {@link https://hono.dev/docs/api/routing#grouping}
   *
   * @param {string} path - base Path
   * @param {Hono} app - other Hono instance
   * @returns {Hono} routed Hono instance
   *
   * @example
   * ```ts
   * const app = new Hono()
   * const app2 = new Hono()
   *
   * app2.get("/user", (c) => c.text("user"))
   * app.route("/api", app2) // GET /api/user
   * ```
   */
  route(path, app2) {
    const subApp = this.basePath(path);
    app2.routes.map((r2) => {
      let handler;
      if (app2.errorHandler === errorHandler) {
        handler = r2.handler;
      } else {
        handler = /* @__PURE__ */ __name(async (c, next) => (await compose([], app2.errorHandler)(c, () => r2.handler(c, next))).res, "handler");
        handler[COMPOSED_HANDLER] = r2.handler;
      }
      subApp.#addRoute(r2.method, r2.path, handler);
    });
    return this;
  }
  /**
   * `.basePath()` allows base paths to be specified.
   *
   * @see {@link https://hono.dev/docs/api/routing#base-path}
   *
   * @param {string} path - base Path
   * @returns {Hono} changed Hono instance
   *
   * @example
   * ```ts
   * const api = new Hono().basePath('/api')
   * ```
   */
  basePath(path) {
    const subApp = this.#clone();
    subApp._basePath = mergePath(this._basePath, path);
    return subApp;
  }
  /**
   * `.onError()` handles an error and returns a customized Response.
   *
   * @see {@link https://hono.dev/docs/api/hono#error-handling}
   *
   * @param {ErrorHandler} handler - request Handler for error
   * @returns {Hono} changed Hono instance
   *
   * @example
   * ```ts
   * app.onError((err, c) => {
   *   console.error(`${err}`)
   *   return c.text('Custom Error Message', 500)
   * })
   * ```
   */
  onError = /* @__PURE__ */ __name((handler) => {
    this.errorHandler = handler;
    return this;
  }, "onError");
  /**
   * `.notFound()` allows you to customize a Not Found Response.
   *
   * @see {@link https://hono.dev/docs/api/hono#not-found}
   *
   * @param {NotFoundHandler} handler - request handler for not-found
   * @returns {Hono} changed Hono instance
   *
   * @example
   * ```ts
   * app.notFound((c) => {
   *   return c.text('Custom 404 Message', 404)
   * })
   * ```
   */
  notFound = /* @__PURE__ */ __name((handler) => {
    this.#notFoundHandler = handler;
    return this;
  }, "notFound");
  /**
   * `.mount()` allows you to mount applications built with other frameworks into your Hono application.
   *
   * @see {@link https://hono.dev/docs/api/hono#mount}
   *
   * @param {string} path - base Path
   * @param {Function} applicationHandler - other Request Handler
   * @param {MountOptions} [options] - options of `.mount()`
   * @returns {Hono} mounted Hono instance
   *
   * @example
   * ```ts
   * import { Router as IttyRouter } from 'itty-router'
   * import { Hono } from 'hono'
   * // Create itty-router application
   * const ittyRouter = IttyRouter()
   * // GET /itty-router/hello
   * ittyRouter.get('/hello', () => new Response('Hello from itty-router'))
   *
   * const app = new Hono()
   * app.mount('/itty-router', ittyRouter.handle)
   * ```
   *
   * @example
   * ```ts
   * const app = new Hono()
   * // Send the request to another application without modification.
   * app.mount('/app', anotherApp, {
   *   replaceRequest: (req) => req,
   * })
   * ```
   */
  mount(path, applicationHandler, options) {
    let replaceRequest;
    let optionHandler;
    if (options) {
      if (typeof options === "function") {
        optionHandler = options;
      } else {
        optionHandler = options.optionHandler;
        if (options.replaceRequest === false) {
          replaceRequest = /* @__PURE__ */ __name((request) => request, "replaceRequest");
        } else {
          replaceRequest = options.replaceRequest;
        }
      }
    }
    const getOptions = optionHandler ? (c) => {
      const options2 = optionHandler(c);
      return Array.isArray(options2) ? options2 : [options2];
    } : (c) => {
      let executionContext = void 0;
      try {
        executionContext = c.executionCtx;
      } catch {
      }
      return [c.env, executionContext];
    };
    replaceRequest ||= (() => {
      const mergedPath = mergePath(this._basePath, path);
      const pathPrefixLength = mergedPath === "/" ? 0 : mergedPath.length;
      return (request) => {
        const url = new URL(request.url);
        url.pathname = url.pathname.slice(pathPrefixLength) || "/";
        return new Request(url, request);
      };
    })();
    const handler = /* @__PURE__ */ __name(async (c, next) => {
      const res = await applicationHandler(replaceRequest(c.req.raw), ...getOptions(c));
      if (res) {
        return res;
      }
      await next();
    }, "handler");
    this.#addRoute(METHOD_NAME_ALL, mergePath(path, "*"), handler);
    return this;
  }
  #addRoute(method, path, handler) {
    method = method.toUpperCase();
    path = mergePath(this._basePath, path);
    const r2 = { basePath: this._basePath, path, method, handler };
    this.router.add(method, path, [handler, r2]);
    this.routes.push(r2);
  }
  #handleError(err, c) {
    if (err instanceof Error) {
      return this.errorHandler(err, c);
    }
    throw err;
  }
  #dispatch(request, executionCtx, env2, method) {
    if (method === "HEAD") {
      return (async () => new Response(null, await this.#dispatch(request, executionCtx, env2, "GET")))();
    }
    const path = this.getPath(request, { env: env2 });
    const matchResult = this.router.match(method, path);
    const c = new Context(request, {
      path,
      matchResult,
      env: env2,
      executionCtx,
      notFoundHandler: this.#notFoundHandler
    });
    if (matchResult[0].length === 1) {
      let res;
      try {
        res = matchResult[0][0][0][0](c, async () => {
          c.res = await this.#notFoundHandler(c);
        });
      } catch (err) {
        return this.#handleError(err, c);
      }
      return res instanceof Promise ? res.then(
        (resolved) => resolved || (c.finalized ? c.res : this.#notFoundHandler(c))
      ).catch((err) => this.#handleError(err, c)) : res ?? this.#notFoundHandler(c);
    }
    const composed = compose(matchResult[0], this.errorHandler, this.#notFoundHandler);
    return (async () => {
      try {
        const context2 = await composed(c);
        if (!context2.finalized) {
          throw new Error(
            "Context is not finalized. Did you forget to return a Response object or `await next()`?"
          );
        }
        return context2.res;
      } catch (err) {
        return this.#handleError(err, c);
      }
    })();
  }
  /**
   * `.fetch()` will be entry point of your app.
   *
   * @see {@link https://hono.dev/docs/api/hono#fetch}
   *
   * @param {Request} request - request Object of request
   * @param {Env} Env - env Object
   * @param {ExecutionContext} - context of execution
   * @returns {Response | Promise<Response>} response of request
   *
   */
  fetch = /* @__PURE__ */ __name((request, ...rest) => {
    return this.#dispatch(request, rest[1], rest[0], request.method);
  }, "fetch");
  /**
   * `.request()` is a useful method for testing.
   * You can pass a URL or pathname to send a GET request.
   * app will return a Response object.
   * ```ts
   * test('GET /hello is ok', async () => {
   *   const res = await app.request('/hello')
   *   expect(res.status).toBe(200)
   * })
   * ```
   * @see https://hono.dev/docs/api/hono#request
   */
  request = /* @__PURE__ */ __name((input, requestInit, Env2, executionCtx) => {
    if (input instanceof Request) {
      return this.fetch(requestInit ? new Request(input, requestInit) : input, Env2, executionCtx);
    }
    input = input.toString();
    return this.fetch(
      new Request(
        /^https?:\/\//.test(input) ? input : `http://localhost${mergePath("/", input)}`,
        requestInit
      ),
      Env2,
      executionCtx
    );
  }, "request");
  /**
   * `.fire()` automatically adds a global fetch event listener.
   * This can be useful for environments that adhere to the Service Worker API, such as non-ES module Cloudflare Workers.
   * @deprecated
   * Use `fire` from `hono/service-worker` instead.
   * ```ts
   * import { Hono } from 'hono'
   * import { fire } from 'hono/service-worker'
   *
   * const app = new Hono()
   * // ...
   * fire(app)
   * ```
   * @see https://hono.dev/docs/api/hono#fire
   * @see https://developer.mozilla.org/en-US/docs/Web/API/Service_Worker_API
   * @see https://developers.cloudflare.com/workers/reference/migrate-to-module-workers/
   */
  fire = /* @__PURE__ */ __name(() => {
    addEventListener("fetch", (event) => {
      event.respondWith(this.#dispatch(event.request, event, void 0, event.request.method));
    });
  }, "fire");
};

// node_modules/hono/dist/router/reg-exp-router/matcher.js
var emptyParam = [];
function match(method, path) {
  const matchers = this.buildAllMatchers();
  const match2 = /* @__PURE__ */ __name(((method2, path2) => {
    const matcher = matchers[method2] || matchers[METHOD_NAME_ALL];
    const staticMatch = matcher[2][path2];
    if (staticMatch) {
      return staticMatch;
    }
    const match3 = path2.match(matcher[0]);
    if (!match3) {
      return [[], emptyParam];
    }
    const index = match3.indexOf("", 1);
    return [matcher[1][index], match3];
  }), "match2");
  this.match = match2;
  return match2(method, path);
}
__name(match, "match");

// node_modules/hono/dist/router/reg-exp-router/node.js
var LABEL_REG_EXP_STR = "[^/]+";
var ONLY_WILDCARD_REG_EXP_STR = ".*";
var TAIL_WILDCARD_REG_EXP_STR = "(?:|/.*)";
var PATH_ERROR = /* @__PURE__ */ Symbol();
var regExpMetaChars = new Set(".\\+*[^]$()");
function compareKey(a3, b) {
  if (a3.length === 1) {
    return b.length === 1 ? a3 < b ? -1 : 1 : -1;
  }
  if (b.length === 1) {
    return 1;
  }
  if (a3 === ONLY_WILDCARD_REG_EXP_STR || a3 === TAIL_WILDCARD_REG_EXP_STR) {
    return 1;
  } else if (b === ONLY_WILDCARD_REG_EXP_STR || b === TAIL_WILDCARD_REG_EXP_STR) {
    return -1;
  }
  if (a3 === LABEL_REG_EXP_STR) {
    return 1;
  } else if (b === LABEL_REG_EXP_STR) {
    return -1;
  }
  return a3.length === b.length ? a3 < b ? -1 : 1 : b.length - a3.length;
}
__name(compareKey, "compareKey");
var Node = class _Node {
  static {
    __name(this, "_Node");
  }
  #index;
  #varIndex;
  #children = /* @__PURE__ */ Object.create(null);
  insert(tokens, index, paramMap, context2, pathErrorCheckOnly) {
    if (tokens.length === 0) {
      if (this.#index !== void 0) {
        throw PATH_ERROR;
      }
      if (pathErrorCheckOnly) {
        return;
      }
      this.#index = index;
      return;
    }
    const [token, ...restTokens] = tokens;
    const pattern = token === "*" ? restTokens.length === 0 ? ["", "", ONLY_WILDCARD_REG_EXP_STR] : ["", "", LABEL_REG_EXP_STR] : token === "/*" ? ["", "", TAIL_WILDCARD_REG_EXP_STR] : token.match(/^\:([^\{\}]+)(?:\{(.+)\})?$/);
    let node;
    if (pattern) {
      const name = pattern[1];
      let regexpStr = pattern[2] || LABEL_REG_EXP_STR;
      if (name && pattern[2]) {
        if (regexpStr === ".*") {
          throw PATH_ERROR;
        }
        regexpStr = regexpStr.replace(/^\((?!\?:)(?=[^)]+\)$)/, "(?:");
        if (/\((?!\?:)/.test(regexpStr)) {
          throw PATH_ERROR;
        }
      }
      node = this.#children[regexpStr];
      if (!node) {
        if (Object.keys(this.#children).some(
          (k) => k !== ONLY_WILDCARD_REG_EXP_STR && k !== TAIL_WILDCARD_REG_EXP_STR
        )) {
          throw PATH_ERROR;
        }
        if (pathErrorCheckOnly) {
          return;
        }
        node = this.#children[regexpStr] = new _Node();
        if (name !== "") {
          node.#varIndex = context2.varIndex++;
        }
      }
      if (!pathErrorCheckOnly && name !== "") {
        paramMap.push([name, node.#varIndex]);
      }
    } else {
      node = this.#children[token];
      if (!node) {
        if (Object.keys(this.#children).some(
          (k) => k.length > 1 && k !== ONLY_WILDCARD_REG_EXP_STR && k !== TAIL_WILDCARD_REG_EXP_STR
        )) {
          throw PATH_ERROR;
        }
        if (pathErrorCheckOnly) {
          return;
        }
        node = this.#children[token] = new _Node();
      }
    }
    node.insert(restTokens, index, paramMap, context2, pathErrorCheckOnly);
  }
  buildRegExpStr() {
    const childKeys = Object.keys(this.#children).sort(compareKey);
    const strList = childKeys.map((k) => {
      const c = this.#children[k];
      return (typeof c.#varIndex === "number" ? `(${k})@${c.#varIndex}` : regExpMetaChars.has(k) ? `\\${k}` : k) + c.buildRegExpStr();
    });
    if (typeof this.#index === "number") {
      strList.unshift(`#${this.#index}`);
    }
    if (strList.length === 0) {
      return "";
    }
    if (strList.length === 1) {
      return strList[0];
    }
    return "(?:" + strList.join("|") + ")";
  }
};

// node_modules/hono/dist/router/reg-exp-router/trie.js
var Trie = class {
  static {
    __name(this, "Trie");
  }
  #context = { varIndex: 0 };
  #root = new Node();
  insert(path, index, pathErrorCheckOnly) {
    const paramAssoc = [];
    const groups = [];
    for (let i3 = 0; ; ) {
      let replaced = false;
      path = path.replace(/\{[^}]+\}/g, (m) => {
        const mark = `@\\${i3}`;
        groups[i3] = [mark, m];
        i3++;
        replaced = true;
        return mark;
      });
      if (!replaced) {
        break;
      }
    }
    const tokens = path.match(/(?::[^\/]+)|(?:\/\*$)|./g) || [];
    for (let i3 = groups.length - 1; i3 >= 0; i3--) {
      const [mark] = groups[i3];
      for (let j = tokens.length - 1; j >= 0; j--) {
        if (tokens[j].indexOf(mark) !== -1) {
          tokens[j] = tokens[j].replace(mark, groups[i3][1]);
          break;
        }
      }
    }
    this.#root.insert(tokens, index, paramAssoc, this.#context, pathErrorCheckOnly);
    return paramAssoc;
  }
  buildRegExp() {
    let regexp = this.#root.buildRegExpStr();
    if (regexp === "") {
      return [/^$/, [], []];
    }
    let captureIndex = 0;
    const indexReplacementMap = [];
    const paramReplacementMap = [];
    regexp = regexp.replace(/#(\d+)|@(\d+)|\.\*\$/g, (_, handlerIndex, paramIndex) => {
      if (handlerIndex !== void 0) {
        indexReplacementMap[++captureIndex] = Number(handlerIndex);
        return "$()";
      }
      if (paramIndex !== void 0) {
        paramReplacementMap[Number(paramIndex)] = ++captureIndex;
        return "";
      }
      return "";
    });
    return [new RegExp(`^${regexp}`), indexReplacementMap, paramReplacementMap];
  }
};

// node_modules/hono/dist/router/reg-exp-router/router.js
var nullMatcher = [/^$/, [], /* @__PURE__ */ Object.create(null)];
var wildcardRegExpCache = /* @__PURE__ */ Object.create(null);
function buildWildcardRegExp(path) {
  return wildcardRegExpCache[path] ??= new RegExp(
    path === "*" ? "" : `^${path.replace(
      /\/\*$|([.\\+*[^\]$()])/g,
      (_, metaChar) => metaChar ? `\\${metaChar}` : "(?:|/.*)"
    )}$`
  );
}
__name(buildWildcardRegExp, "buildWildcardRegExp");
function clearWildcardRegExpCache() {
  wildcardRegExpCache = /* @__PURE__ */ Object.create(null);
}
__name(clearWildcardRegExpCache, "clearWildcardRegExpCache");
function buildMatcherFromPreprocessedRoutes(routes) {
  const trie = new Trie();
  const handlerData = [];
  if (routes.length === 0) {
    return nullMatcher;
  }
  const routesWithStaticPathFlag = routes.map(
    (route) => [!/\*|\/:/.test(route[0]), ...route]
  ).sort(
    ([isStaticA, pathA], [isStaticB, pathB]) => isStaticA ? 1 : isStaticB ? -1 : pathA.length - pathB.length
  );
  const staticMap = /* @__PURE__ */ Object.create(null);
  for (let i3 = 0, j = -1, len = routesWithStaticPathFlag.length; i3 < len; i3++) {
    const [pathErrorCheckOnly, path, handlers] = routesWithStaticPathFlag[i3];
    if (pathErrorCheckOnly) {
      staticMap[path] = [handlers.map(([h3]) => [h3, /* @__PURE__ */ Object.create(null)]), emptyParam];
    } else {
      j++;
    }
    let paramAssoc;
    try {
      paramAssoc = trie.insert(path, j, pathErrorCheckOnly);
    } catch (e3) {
      throw e3 === PATH_ERROR ? new UnsupportedPathError(path) : e3;
    }
    if (pathErrorCheckOnly) {
      continue;
    }
    handlerData[j] = handlers.map(([h3, paramCount]) => {
      const paramIndexMap = /* @__PURE__ */ Object.create(null);
      paramCount -= 1;
      for (; paramCount >= 0; paramCount--) {
        const [key, value] = paramAssoc[paramCount];
        paramIndexMap[key] = value;
      }
      return [h3, paramIndexMap];
    });
  }
  const [regexp, indexReplacementMap, paramReplacementMap] = trie.buildRegExp();
  for (let i3 = 0, len = handlerData.length; i3 < len; i3++) {
    for (let j = 0, len2 = handlerData[i3].length; j < len2; j++) {
      const map = handlerData[i3][j]?.[1];
      if (!map) {
        continue;
      }
      const keys = Object.keys(map);
      for (let k = 0, len3 = keys.length; k < len3; k++) {
        map[keys[k]] = paramReplacementMap[map[keys[k]]];
      }
    }
  }
  const handlerMap = [];
  for (const i3 in indexReplacementMap) {
    handlerMap[i3] = handlerData[indexReplacementMap[i3]];
  }
  return [regexp, handlerMap, staticMap];
}
__name(buildMatcherFromPreprocessedRoutes, "buildMatcherFromPreprocessedRoutes");
function findMiddleware(middleware, path) {
  if (!middleware) {
    return void 0;
  }
  for (const k of Object.keys(middleware).sort((a3, b) => b.length - a3.length)) {
    if (buildWildcardRegExp(k).test(path)) {
      return [...middleware[k]];
    }
  }
  return void 0;
}
__name(findMiddleware, "findMiddleware");
var RegExpRouter = class {
  static {
    __name(this, "RegExpRouter");
  }
  name = "RegExpRouter";
  #middleware;
  #routes;
  constructor() {
    this.#middleware = { [METHOD_NAME_ALL]: /* @__PURE__ */ Object.create(null) };
    this.#routes = { [METHOD_NAME_ALL]: /* @__PURE__ */ Object.create(null) };
  }
  add(method, path, handler) {
    const middleware = this.#middleware;
    const routes = this.#routes;
    if (!middleware || !routes) {
      throw new Error(MESSAGE_MATCHER_IS_ALREADY_BUILT);
    }
    if (!middleware[method]) {
      ;
      [middleware, routes].forEach((handlerMap) => {
        handlerMap[method] = /* @__PURE__ */ Object.create(null);
        Object.keys(handlerMap[METHOD_NAME_ALL]).forEach((p) => {
          handlerMap[method][p] = [...handlerMap[METHOD_NAME_ALL][p]];
        });
      });
    }
    if (path === "/*") {
      path = "*";
    }
    const paramCount = (path.match(/\/:/g) || []).length;
    if (/\*$/.test(path)) {
      const re = buildWildcardRegExp(path);
      if (method === METHOD_NAME_ALL) {
        Object.keys(middleware).forEach((m) => {
          middleware[m][path] ||= findMiddleware(middleware[m], path) || findMiddleware(middleware[METHOD_NAME_ALL], path) || [];
        });
      } else {
        middleware[method][path] ||= findMiddleware(middleware[method], path) || findMiddleware(middleware[METHOD_NAME_ALL], path) || [];
      }
      Object.keys(middleware).forEach((m) => {
        if (method === METHOD_NAME_ALL || method === m) {
          Object.keys(middleware[m]).forEach((p) => {
            re.test(p) && middleware[m][p].push([handler, paramCount]);
          });
        }
      });
      Object.keys(routes).forEach((m) => {
        if (method === METHOD_NAME_ALL || method === m) {
          Object.keys(routes[m]).forEach(
            (p) => re.test(p) && routes[m][p].push([handler, paramCount])
          );
        }
      });
      return;
    }
    const paths = checkOptionalParameter(path) || [path];
    for (let i3 = 0, len = paths.length; i3 < len; i3++) {
      const path2 = paths[i3];
      Object.keys(routes).forEach((m) => {
        if (method === METHOD_NAME_ALL || method === m) {
          routes[m][path2] ||= [
            ...findMiddleware(middleware[m], path2) || findMiddleware(middleware[METHOD_NAME_ALL], path2) || []
          ];
          routes[m][path2].push([handler, paramCount - len + i3 + 1]);
        }
      });
    }
  }
  match = match;
  buildAllMatchers() {
    const matchers = /* @__PURE__ */ Object.create(null);
    Object.keys(this.#routes).concat(Object.keys(this.#middleware)).forEach((method) => {
      matchers[method] ||= this.#buildMatcher(method);
    });
    this.#middleware = this.#routes = void 0;
    clearWildcardRegExpCache();
    return matchers;
  }
  #buildMatcher(method) {
    const routes = [];
    let hasOwnRoute = method === METHOD_NAME_ALL;
    [this.#middleware, this.#routes].forEach((r2) => {
      const ownRoute = r2[method] ? Object.keys(r2[method]).map((path) => [path, r2[method][path]]) : [];
      if (ownRoute.length !== 0) {
        hasOwnRoute ||= true;
        routes.push(...ownRoute);
      } else if (method !== METHOD_NAME_ALL) {
        routes.push(
          ...Object.keys(r2[METHOD_NAME_ALL]).map((path) => [path, r2[METHOD_NAME_ALL][path]])
        );
      }
    });
    if (!hasOwnRoute) {
      return null;
    } else {
      return buildMatcherFromPreprocessedRoutes(routes);
    }
  }
};

// node_modules/hono/dist/router/smart-router/router.js
var SmartRouter = class {
  static {
    __name(this, "SmartRouter");
  }
  name = "SmartRouter";
  #routers = [];
  #routes = [];
  constructor(init) {
    this.#routers = init.routers;
  }
  add(method, path, handler) {
    if (!this.#routes) {
      throw new Error(MESSAGE_MATCHER_IS_ALREADY_BUILT);
    }
    this.#routes.push([method, path, handler]);
  }
  match(method, path) {
    if (!this.#routes) {
      throw new Error("Fatal error");
    }
    const routers = this.#routers;
    const routes = this.#routes;
    const len = routers.length;
    let i3 = 0;
    let res;
    for (; i3 < len; i3++) {
      const router = routers[i3];
      try {
        for (let i22 = 0, len2 = routes.length; i22 < len2; i22++) {
          router.add(...routes[i22]);
        }
        res = router.match(method, path);
      } catch (e3) {
        if (e3 instanceof UnsupportedPathError) {
          continue;
        }
        throw e3;
      }
      this.match = router.match.bind(router);
      this.#routers = [router];
      this.#routes = void 0;
      break;
    }
    if (i3 === len) {
      throw new Error("Fatal error");
    }
    this.name = `SmartRouter + ${this.activeRouter.name}`;
    return res;
  }
  get activeRouter() {
    if (this.#routes || this.#routers.length !== 1) {
      throw new Error("No active router has been determined yet.");
    }
    return this.#routers[0];
  }
};

// node_modules/hono/dist/router/trie-router/node.js
var emptyParams = /* @__PURE__ */ Object.create(null);
var Node2 = class _Node2 {
  static {
    __name(this, "_Node");
  }
  #methods;
  #children;
  #patterns;
  #order = 0;
  #params = emptyParams;
  constructor(method, handler, children) {
    this.#children = children || /* @__PURE__ */ Object.create(null);
    this.#methods = [];
    if (method && handler) {
      const m = /* @__PURE__ */ Object.create(null);
      m[method] = { handler, possibleKeys: [], score: 0 };
      this.#methods = [m];
    }
    this.#patterns = [];
  }
  insert(method, path, handler) {
    this.#order = ++this.#order;
    let curNode = this;
    const parts = splitRoutingPath(path);
    const possibleKeys = [];
    for (let i3 = 0, len = parts.length; i3 < len; i3++) {
      const p = parts[i3];
      const nextP = parts[i3 + 1];
      const pattern = getPattern(p, nextP);
      const key = Array.isArray(pattern) ? pattern[0] : p;
      if (key in curNode.#children) {
        curNode = curNode.#children[key];
        if (pattern) {
          possibleKeys.push(pattern[1]);
        }
        continue;
      }
      curNode.#children[key] = new _Node2();
      if (pattern) {
        curNode.#patterns.push(pattern);
        possibleKeys.push(pattern[1]);
      }
      curNode = curNode.#children[key];
    }
    curNode.#methods.push({
      [method]: {
        handler,
        possibleKeys: possibleKeys.filter((v, i3, a3) => a3.indexOf(v) === i3),
        score: this.#order
      }
    });
    return curNode;
  }
  #getHandlerSets(node, method, nodeParams, params) {
    const handlerSets = [];
    for (let i3 = 0, len = node.#methods.length; i3 < len; i3++) {
      const m = node.#methods[i3];
      const handlerSet = m[method] || m[METHOD_NAME_ALL];
      const processedSet = {};
      if (handlerSet !== void 0) {
        handlerSet.params = /* @__PURE__ */ Object.create(null);
        handlerSets.push(handlerSet);
        if (nodeParams !== emptyParams || params && params !== emptyParams) {
          for (let i22 = 0, len2 = handlerSet.possibleKeys.length; i22 < len2; i22++) {
            const key = handlerSet.possibleKeys[i22];
            const processed = processedSet[handlerSet.score];
            handlerSet.params[key] = params?.[key] && !processed ? params[key] : nodeParams[key] ?? params?.[key];
            processedSet[handlerSet.score] = true;
          }
        }
      }
    }
    return handlerSets;
  }
  search(method, path) {
    const handlerSets = [];
    this.#params = emptyParams;
    const curNode = this;
    let curNodes = [curNode];
    const parts = splitPath(path);
    const curNodesQueue = [];
    for (let i3 = 0, len = parts.length; i3 < len; i3++) {
      const part = parts[i3];
      const isLast = i3 === len - 1;
      const tempNodes = [];
      for (let j = 0, len2 = curNodes.length; j < len2; j++) {
        const node = curNodes[j];
        const nextNode = node.#children[part];
        if (nextNode) {
          nextNode.#params = node.#params;
          if (isLast) {
            if (nextNode.#children["*"]) {
              handlerSets.push(
                ...this.#getHandlerSets(nextNode.#children["*"], method, node.#params)
              );
            }
            handlerSets.push(...this.#getHandlerSets(nextNode, method, node.#params));
          } else {
            tempNodes.push(nextNode);
          }
        }
        for (let k = 0, len3 = node.#patterns.length; k < len3; k++) {
          const pattern = node.#patterns[k];
          const params = node.#params === emptyParams ? {} : { ...node.#params };
          if (pattern === "*") {
            const astNode = node.#children["*"];
            if (astNode) {
              handlerSets.push(...this.#getHandlerSets(astNode, method, node.#params));
              astNode.#params = params;
              tempNodes.push(astNode);
            }
            continue;
          }
          const [key, name, matcher] = pattern;
          if (!part && !(matcher instanceof RegExp)) {
            continue;
          }
          const child = node.#children[key];
          const restPathString = parts.slice(i3).join("/");
          if (matcher instanceof RegExp) {
            const m = matcher.exec(restPathString);
            if (m) {
              params[name] = m[0];
              handlerSets.push(...this.#getHandlerSets(child, method, node.#params, params));
              if (Object.keys(child.#children).length) {
                child.#params = params;
                const componentCount = m[0].match(/\//)?.length ?? 0;
                const targetCurNodes = curNodesQueue[componentCount] ||= [];
                targetCurNodes.push(child);
              }
              continue;
            }
          }
          if (matcher === true || matcher.test(part)) {
            params[name] = part;
            if (isLast) {
              handlerSets.push(...this.#getHandlerSets(child, method, params, node.#params));
              if (child.#children["*"]) {
                handlerSets.push(
                  ...this.#getHandlerSets(child.#children["*"], method, params, node.#params)
                );
              }
            } else {
              child.#params = params;
              tempNodes.push(child);
            }
          }
        }
      }
      curNodes = tempNodes.concat(curNodesQueue.shift() ?? []);
    }
    if (handlerSets.length > 1) {
      handlerSets.sort((a3, b) => {
        return a3.score - b.score;
      });
    }
    return [handlerSets.map(({ handler, params }) => [handler, params])];
  }
};

// node_modules/hono/dist/router/trie-router/router.js
var TrieRouter = class {
  static {
    __name(this, "TrieRouter");
  }
  name = "TrieRouter";
  #node;
  constructor() {
    this.#node = new Node2();
  }
  add(method, path, handler) {
    const results = checkOptionalParameter(path);
    if (results) {
      for (let i3 = 0, len = results.length; i3 < len; i3++) {
        this.#node.insert(method, results[i3], handler);
      }
      return;
    }
    this.#node.insert(method, path, handler);
  }
  match(method, path) {
    return this.#node.search(method, path);
  }
};

// node_modules/hono/dist/hono.js
var Hono2 = class extends Hono {
  static {
    __name(this, "Hono");
  }
  /**
   * Creates an instance of the Hono class.
   *
   * @param options - Optional configuration options for the Hono instance.
   */
  constructor(options = {}) {
    super(options);
    this.router = options.router ?? new SmartRouter({
      routers: [new RegExpRouter(), new TrieRouter()]
    });
  }
};

// node_modules/hono/dist/middleware/cors/index.js
var cors = /* @__PURE__ */ __name((options) => {
  const defaults = {
    origin: "*",
    allowMethods: ["GET", "HEAD", "PUT", "POST", "DELETE", "PATCH"],
    allowHeaders: [],
    exposeHeaders: []
  };
  const opts = {
    ...defaults,
    ...options
  };
  const findAllowOrigin = ((optsOrigin) => {
    if (typeof optsOrigin === "string") {
      if (optsOrigin === "*") {
        return () => optsOrigin;
      } else {
        return (origin) => optsOrigin === origin ? origin : null;
      }
    } else if (typeof optsOrigin === "function") {
      return optsOrigin;
    } else {
      return (origin) => optsOrigin.includes(origin) ? origin : null;
    }
  })(opts.origin);
  const findAllowMethods = ((optsAllowMethods) => {
    if (typeof optsAllowMethods === "function") {
      return optsAllowMethods;
    } else if (Array.isArray(optsAllowMethods)) {
      return () => optsAllowMethods;
    } else {
      return () => [];
    }
  })(opts.allowMethods);
  return /* @__PURE__ */ __name(async function cors2(c, next) {
    function set(key, value) {
      c.res.headers.set(key, value);
    }
    __name(set, "set");
    const allowOrigin = await findAllowOrigin(c.req.header("origin") || "", c);
    if (allowOrigin) {
      set("Access-Control-Allow-Origin", allowOrigin);
    }
    if (opts.credentials) {
      set("Access-Control-Allow-Credentials", "true");
    }
    if (opts.exposeHeaders?.length) {
      set("Access-Control-Expose-Headers", opts.exposeHeaders.join(","));
    }
    if (c.req.method === "OPTIONS") {
      if (opts.origin !== "*") {
        set("Vary", "Origin");
      }
      if (opts.maxAge != null) {
        set("Access-Control-Max-Age", opts.maxAge.toString());
      }
      const allowMethods = await findAllowMethods(c.req.header("origin") || "", c);
      if (allowMethods.length) {
        set("Access-Control-Allow-Methods", allowMethods.join(","));
      }
      let headers = opts.allowHeaders;
      if (!headers?.length) {
        const requestHeaders = c.req.header("Access-Control-Request-Headers");
        if (requestHeaders) {
          headers = requestHeaders.split(/\s*,\s*/);
        }
      }
      if (headers?.length) {
        set("Access-Control-Allow-Headers", headers.join(","));
        c.res.headers.append("Vary", "Access-Control-Request-Headers");
      }
      c.res.headers.delete("Content-Length");
      c.res.headers.delete("Content-Type");
      return new Response(null, {
        headers: c.res.headers,
        status: 204,
        statusText: "No Content"
      });
    }
    await next();
    if (opts.origin !== "*") {
      c.header("Vary", "Origin", { append: true });
    }
  }, "cors2");
}, "cors");

// node_modules/node-fetch-native-with-agent/dist/native.mjs
var e = globalThis.Blob;
var o = globalThis.File;
var a = globalThis.FormData;
var s = globalThis.Headers;
var t = globalThis.Request;
var h = globalThis.Response;
var i = globalThis.AbortController;
var l = globalThis.fetch || (() => {
  throw new Error("[node-fetch-native] Failed to fetch: `globalThis.fetch` is not available!");
});

// node_modules/node-fetch-native-with-agent/dist/agent-stub.mjs
var o2 = Object.defineProperty;
var e2 = /* @__PURE__ */ __name((t2, c) => o2(t2, "name", { value: c, configurable: true }), "e");
var i2 = Object.defineProperty;
var r = e2((t2, c) => i2(t2, "name", { value: c, configurable: true }), "e");
function a2() {
  return { agent: void 0, dispatcher: void 0 };
}
__name(a2, "a");
e2(a2, "createAgent"), r(a2, "createAgent");
function n() {
  return globalThis.fetch;
}
__name(n, "n");
e2(n, "createFetch"), r(n, "createFetch");
var h2 = globalThis.fetch;

// node_modules/node-appwrite/dist/query.mjs
var _Query = class _Query2 {
  static {
    __name(this, "_Query");
  }
  /**
   * Constructor for Query class.
   *
   * @param {string} method
   * @param {AttributesTypes} attribute
   * @param {QueryTypes} values
   */
  constructor(method, attribute, values) {
    this.method = method;
    this.attribute = attribute;
    if (values !== void 0) {
      if (Array.isArray(values)) {
        this.values = values;
      } else {
        this.values = [values];
      }
    }
  }
  /**
   * Convert the query object to a JSON string.
   *
   * @returns {string}
   */
  toString() {
    return JSON.stringify({
      method: this.method,
      attribute: this.attribute,
      values: this.values
    });
  }
};
_Query.equal = (attribute, value) => new _Query("equal", attribute, value).toString();
_Query.notEqual = (attribute, value) => new _Query("notEqual", attribute, value).toString();
_Query.lessThan = (attribute, value) => new _Query("lessThan", attribute, value).toString();
_Query.lessThanEqual = (attribute, value) => new _Query("lessThanEqual", attribute, value).toString();
_Query.greaterThan = (attribute, value) => new _Query("greaterThan", attribute, value).toString();
_Query.greaterThanEqual = (attribute, value) => new _Query("greaterThanEqual", attribute, value).toString();
_Query.isNull = (attribute) => new _Query("isNull", attribute).toString();
_Query.isNotNull = (attribute) => new _Query("isNotNull", attribute).toString();
_Query.between = (attribute, start, end) => new _Query("between", attribute, [start, end]).toString();
_Query.startsWith = (attribute, value) => new _Query("startsWith", attribute, value).toString();
_Query.endsWith = (attribute, value) => new _Query("endsWith", attribute, value).toString();
_Query.select = (attributes) => new _Query("select", void 0, attributes).toString();
_Query.search = (attribute, value) => new _Query("search", attribute, value).toString();
_Query.orderDesc = (attribute) => new _Query("orderDesc", attribute).toString();
_Query.orderAsc = (attribute) => new _Query("orderAsc", attribute).toString();
_Query.orderRandom = () => new _Query("orderRandom").toString();
_Query.cursorAfter = (documentId) => new _Query("cursorAfter", void 0, documentId).toString();
_Query.cursorBefore = (documentId) => new _Query("cursorBefore", void 0, documentId).toString();
_Query.limit = (limit) => new _Query("limit", void 0, limit).toString();
_Query.offset = (offset) => new _Query("offset", void 0, offset).toString();
_Query.contains = (attribute, value) => new _Query("contains", attribute, value).toString();
_Query.notContains = (attribute, value) => new _Query("notContains", attribute, value).toString();
_Query.notSearch = (attribute, value) => new _Query("notSearch", attribute, value).toString();
_Query.notBetween = (attribute, start, end) => new _Query("notBetween", attribute, [start, end]).toString();
_Query.notStartsWith = (attribute, value) => new _Query("notStartsWith", attribute, value).toString();
_Query.notEndsWith = (attribute, value) => new _Query("notEndsWith", attribute, value).toString();
_Query.createdBefore = (value) => _Query.lessThan("$createdAt", value);
_Query.createdAfter = (value) => _Query.greaterThan("$createdAt", value);
_Query.createdBetween = (start, end) => _Query.between("$createdAt", start, end);
_Query.updatedBefore = (value) => _Query.lessThan("$updatedAt", value);
_Query.updatedAfter = (value) => _Query.greaterThan("$updatedAt", value);
_Query.updatedBetween = (start, end) => _Query.between("$updatedAt", start, end);
_Query.or = (queries) => new _Query("or", void 0, queries.map((query) => JSON.parse(query))).toString();
_Query.and = (queries) => new _Query("and", void 0, queries.map((query) => JSON.parse(query))).toString();
_Query.distanceEqual = (attribute, values, distance, meters = true) => new _Query("distanceEqual", attribute, [[values, distance, meters]]).toString();
_Query.distanceNotEqual = (attribute, values, distance, meters = true) => new _Query("distanceNotEqual", attribute, [[values, distance, meters]]).toString();
_Query.distanceGreaterThan = (attribute, values, distance, meters = true) => new _Query("distanceGreaterThan", attribute, [[values, distance, meters]]).toString();
_Query.distanceLessThan = (attribute, values, distance, meters = true) => new _Query("distanceLessThan", attribute, [[values, distance, meters]]).toString();
_Query.intersects = (attribute, values) => new _Query("intersects", attribute, [values]).toString();
_Query.notIntersects = (attribute, values) => new _Query("notIntersects", attribute, [values]).toString();
_Query.crosses = (attribute, values) => new _Query("crosses", attribute, [values]).toString();
_Query.notCrosses = (attribute, values) => new _Query("notCrosses", attribute, [values]).toString();
_Query.overlaps = (attribute, values) => new _Query("overlaps", attribute, [values]).toString();
_Query.notOverlaps = (attribute, values) => new _Query("notOverlaps", attribute, [values]).toString();
_Query.touches = (attribute, values) => new _Query("touches", attribute, [values]).toString();
_Query.notTouches = (attribute, values) => new _Query("notTouches", attribute, [values]).toString();
var Query = _Query;

// node_modules/node-appwrite/dist/client.mjs
var AppwriteException = class extends Error {
  static {
    __name(this, "AppwriteException");
  }
  constructor(message, code = 0, type = "", response = "") {
    super(message);
    this.name = "AppwriteException";
    this.message = message;
    this.code = code;
    this.type = type;
    this.response = response;
  }
};
function getUserAgent() {
  let ua = "AppwriteNodeJSSDK/21.1.0";
  const platform2 = [];
  if (typeof process !== "undefined") {
    if (typeof process.platform === "string")
      platform2.push(process.platform);
    if (typeof process.arch === "string")
      platform2.push(process.arch);
  }
  if (platform2.length > 0) {
    ua += ` (${platform2.join("; ")})`;
  }
  if (typeof navigator !== "undefined" && true) {
    ua += ` ${"Cloudflare-Workers"}`;
  } else if (typeof globalThis.EdgeRuntime === "string") {
    ua += ` EdgeRuntime`;
  } else if (typeof process !== "undefined" && typeof process.version === "string") {
    ua += ` Node.js/${process.version}`;
  }
  return ua;
}
__name(getUserAgent, "getUserAgent");
var _Client = class _Client2 {
  static {
    __name(this, "_Client");
  }
  constructor() {
    this.config = {
      endpoint: "https://cloud.appwrite.io/v1",
      selfSigned: false,
      project: "",
      key: "",
      jwt: "",
      locale: "",
      session: "",
      forwardeduseragent: ""
    };
    this.headers = {
      "x-sdk-name": "Node.js",
      "x-sdk-platform": "server",
      "x-sdk-language": "nodejs",
      "x-sdk-version": "21.1.0",
      "user-agent": getUserAgent(),
      "X-Appwrite-Response-Format": "1.8.0"
    };
  }
  /**
   * Set Endpoint
   *
   * Your project endpoint
   *
   * @param {string} endpoint
   *
   * @returns {this}
   */
  setEndpoint(endpoint) {
    if (!endpoint.startsWith("http://") && !endpoint.startsWith("https://")) {
      throw new AppwriteException("Invalid endpoint URL: " + endpoint);
    }
    this.config.endpoint = endpoint;
    return this;
  }
  /**
   * Set self-signed
   *
   * @param {boolean} selfSigned
   *
   * @returns {this}
   */
  setSelfSigned(selfSigned) {
    if (typeof globalThis.EdgeRuntime !== "undefined") {
      console.warn("setSelfSigned is not supported in edge runtimes.");
    }
    this.config.selfSigned = selfSigned;
    return this;
  }
  /**
   * Add header
   *
   * @param {string} header
   * @param {string} value
   *
   * @returns {this}
   */
  addHeader(header, value) {
    this.headers[header.toLowerCase()] = value;
    return this;
  }
  /**
   * Set Project
   *
   * Your project ID
   *
   * @param value string
   *
   * @return {this}
   */
  setProject(value) {
    this.headers["X-Appwrite-Project"] = value;
    this.config.project = value;
    return this;
  }
  /**
   * Set Key
   *
   * Your secret API key
   *
   * @param value string
   *
   * @return {this}
   */
  setKey(value) {
    this.headers["X-Appwrite-Key"] = value;
    this.config.key = value;
    return this;
  }
  /**
   * Set JWT
   *
   * Your secret JSON Web Token
   *
   * @param value string
   *
   * @return {this}
   */
  setJWT(value) {
    this.headers["X-Appwrite-JWT"] = value;
    this.config.jwt = value;
    return this;
  }
  /**
   * Set Locale
   *
   * @param value string
   *
   * @return {this}
   */
  setLocale(value) {
    this.headers["X-Appwrite-Locale"] = value;
    this.config.locale = value;
    return this;
  }
  /**
   * Set Session
   *
   * The user session to authenticate with
   *
   * @param value string
   *
   * @return {this}
   */
  setSession(value) {
    this.headers["X-Appwrite-Session"] = value;
    this.config.session = value;
    return this;
  }
  /**
   * Set ForwardedUserAgent
   *
   * The user agent string of the client that made the request
   *
   * @param value string
   *
   * @return {this}
   */
  setForwardedUserAgent(value) {
    this.headers["X-Forwarded-User-Agent"] = value;
    this.config.forwardeduseragent = value;
    return this;
  }
  prepareRequest(method, url, headers = {}, params = {}) {
    method = method.toUpperCase();
    headers = Object.assign({}, this.headers, headers);
    let options = {
      method,
      headers,
      ...a2(this.config.endpoint, { rejectUnauthorized: !this.config.selfSigned })
    };
    if (method === "GET") {
      for (const [key, value] of Object.entries(_Client2.flatten(params))) {
        url.searchParams.append(key, value);
      }
    } else {
      switch (headers["content-type"]) {
        case "application/json":
          options.body = JSON.stringify(params);
          break;
        case "multipart/form-data":
          const formData = new a();
          for (const [key, value] of Object.entries(params)) {
            if (value instanceof o) {
              formData.append(key, value, value.name);
            } else if (Array.isArray(value)) {
              for (const nestedValue of value) {
                formData.append(`${key}[]`, nestedValue);
              }
            } else {
              formData.append(key, value);
            }
          }
          options.body = formData;
          delete headers["content-type"];
          break;
      }
    }
    return { uri: url.toString(), options };
  }
  async chunkedUpload(method, url, headers = {}, originalPayload = {}, onProgress) {
    const [fileParam, file] = Object.entries(originalPayload).find(([_, value]) => value instanceof o) ?? [];
    if (!file || !fileParam) {
      throw new Error("File not found in payload");
    }
    if (file.size <= _Client2.CHUNK_SIZE) {
      return await this.call(method, url, headers, originalPayload);
    }
    let start = 0;
    let response = null;
    while (start < file.size) {
      let end = start + _Client2.CHUNK_SIZE;
      if (end >= file.size) {
        end = file.size;
      }
      headers["content-range"] = `bytes ${start}-${end - 1}/${file.size}`;
      const chunk = file.slice(start, end);
      let payload = { ...originalPayload };
      payload[fileParam] = new o([chunk], file.name);
      response = await this.call(method, url, headers, payload);
      if (onProgress && typeof onProgress === "function") {
        onProgress({
          $id: response.$id,
          progress: Math.round(end / file.size * 100),
          sizeUploaded: end,
          chunksTotal: Math.ceil(file.size / _Client2.CHUNK_SIZE),
          chunksUploaded: Math.ceil(end / _Client2.CHUNK_SIZE)
        });
      }
      if (response && response.$id) {
        headers["x-appwrite-id"] = response.$id;
      }
      start = end;
    }
    return response;
  }
  async ping() {
    return this.call("GET", new URL(this.config.endpoint + "/ping"));
  }
  async redirect(method, url, headers = {}, params = {}) {
    const { uri, options } = this.prepareRequest(method, url, headers, params);
    const response = await l(uri, {
      ...options,
      redirect: "manual"
    });
    if (response.status !== 301 && response.status !== 302) {
      throw new AppwriteException("Invalid redirect", response.status);
    }
    return response.headers.get("location") || "";
  }
  async call(method, url, headers = {}, params = {}, responseType = "json") {
    var _a, _b;
    const { uri, options } = this.prepareRequest(method, url, headers, params);
    let data = null;
    const response = await l(uri, options);
    const warnings = response.headers.get("x-appwrite-warning");
    if (warnings) {
      warnings.split(";").forEach((warning) => console.warn("Warning: " + warning));
    }
    if ((_a = response.headers.get("content-type")) == null ? void 0 : _a.includes("application/json")) {
      data = await response.json();
    } else if (responseType === "arrayBuffer") {
      data = await response.arrayBuffer();
    } else {
      data = {
        message: await response.text()
      };
    }
    if (400 <= response.status) {
      let responseText = "";
      if (((_b = response.headers.get("content-type")) == null ? void 0 : _b.includes("application/json")) || responseType === "arrayBuffer") {
        responseText = JSON.stringify(data);
      } else {
        responseText = data == null ? void 0 : data.message;
      }
      throw new AppwriteException(data == null ? void 0 : data.message, response.status, data == null ? void 0 : data.type, responseText);
    }
    return data;
  }
  static flatten(data, prefix = "") {
    let output = {};
    for (const [key, value] of Object.entries(data)) {
      let finalKey = prefix ? prefix + "[" + key + "]" : key;
      if (Array.isArray(value)) {
        output = { ...output, ..._Client2.flatten(value, finalKey) };
      } else {
        output[finalKey] = value;
      }
    }
    return output;
  }
};
_Client.CHUNK_SIZE = 1024 * 1024 * 5;
var Client = _Client;

// node_modules/node-appwrite/dist/services/databases.mjs
var Databases = class {
  static {
    __name(this, "Databases");
  }
  constructor(client) {
    this.client = client;
  }
  list(paramsOrFirst, ...rest) {
    let params;
    if (!paramsOrFirst || paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        queries: paramsOrFirst,
        search: rest[0],
        total: rest[1]
      };
    }
    const queries = params.queries;
    const search = params.search;
    const total = params.total;
    const apiPath = "/databases";
    const payload = {};
    if (typeof queries !== "undefined") {
      payload["queries"] = queries;
    }
    if (typeof search !== "undefined") {
      payload["search"] = search;
    }
    if (typeof total !== "undefined") {
      payload["total"] = total;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {};
    return this.client.call(
      "get",
      uri,
      apiHeaders,
      payload
    );
  }
  create(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        name: rest[0],
        enabled: rest[1]
      };
    }
    const databaseId = params.databaseId;
    const name = params.name;
    const enabled = params.enabled;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof name === "undefined") {
      throw new AppwriteException('Missing required parameter: "name"');
    }
    const apiPath = "/databases";
    const payload = {};
    if (typeof databaseId !== "undefined") {
      payload["databaseId"] = databaseId;
    }
    if (typeof name !== "undefined") {
      payload["name"] = name;
    }
    if (typeof enabled !== "undefined") {
      payload["enabled"] = enabled;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "post",
      uri,
      apiHeaders,
      payload
    );
  }
  listTransactions(paramsOrFirst) {
    let params;
    if (!paramsOrFirst || paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        queries: paramsOrFirst
      };
    }
    const queries = params.queries;
    const apiPath = "/databases/transactions";
    const payload = {};
    if (typeof queries !== "undefined") {
      payload["queries"] = queries;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {};
    return this.client.call(
      "get",
      uri,
      apiHeaders,
      payload
    );
  }
  createTransaction(paramsOrFirst) {
    let params;
    if (!paramsOrFirst || paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        ttl: paramsOrFirst
      };
    }
    const ttl = params.ttl;
    const apiPath = "/databases/transactions";
    const payload = {};
    if (typeof ttl !== "undefined") {
      payload["ttl"] = ttl;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "post",
      uri,
      apiHeaders,
      payload
    );
  }
  getTransaction(paramsOrFirst) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        transactionId: paramsOrFirst
      };
    }
    const transactionId = params.transactionId;
    if (typeof transactionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "transactionId"');
    }
    const apiPath = "/databases/transactions/{transactionId}".replace("{transactionId}", transactionId);
    const payload = {};
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {};
    return this.client.call(
      "get",
      uri,
      apiHeaders,
      payload
    );
  }
  updateTransaction(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        transactionId: paramsOrFirst,
        commit: rest[0],
        rollback: rest[1]
      };
    }
    const transactionId = params.transactionId;
    const commit = params.commit;
    const rollback = params.rollback;
    if (typeof transactionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "transactionId"');
    }
    const apiPath = "/databases/transactions/{transactionId}".replace("{transactionId}", transactionId);
    const payload = {};
    if (typeof commit !== "undefined") {
      payload["commit"] = commit;
    }
    if (typeof rollback !== "undefined") {
      payload["rollback"] = rollback;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "patch",
      uri,
      apiHeaders,
      payload
    );
  }
  deleteTransaction(paramsOrFirst) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        transactionId: paramsOrFirst
      };
    }
    const transactionId = params.transactionId;
    if (typeof transactionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "transactionId"');
    }
    const apiPath = "/databases/transactions/{transactionId}".replace("{transactionId}", transactionId);
    const payload = {};
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "delete",
      uri,
      apiHeaders,
      payload
    );
  }
  createOperations(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        transactionId: paramsOrFirst,
        operations: rest[0]
      };
    }
    const transactionId = params.transactionId;
    const operations = params.operations;
    if (typeof transactionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "transactionId"');
    }
    const apiPath = "/databases/transactions/{transactionId}/operations".replace("{transactionId}", transactionId);
    const payload = {};
    if (typeof operations !== "undefined") {
      payload["operations"] = operations;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "post",
      uri,
      apiHeaders,
      payload
    );
  }
  get(paramsOrFirst) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst
      };
    }
    const databaseId = params.databaseId;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    const apiPath = "/databases/{databaseId}".replace("{databaseId}", databaseId);
    const payload = {};
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {};
    return this.client.call(
      "get",
      uri,
      apiHeaders,
      payload
    );
  }
  update(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        name: rest[0],
        enabled: rest[1]
      };
    }
    const databaseId = params.databaseId;
    const name = params.name;
    const enabled = params.enabled;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof name === "undefined") {
      throw new AppwriteException('Missing required parameter: "name"');
    }
    const apiPath = "/databases/{databaseId}".replace("{databaseId}", databaseId);
    const payload = {};
    if (typeof name !== "undefined") {
      payload["name"] = name;
    }
    if (typeof enabled !== "undefined") {
      payload["enabled"] = enabled;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "put",
      uri,
      apiHeaders,
      payload
    );
  }
  delete(paramsOrFirst) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst
      };
    }
    const databaseId = params.databaseId;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    const apiPath = "/databases/{databaseId}".replace("{databaseId}", databaseId);
    const payload = {};
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "delete",
      uri,
      apiHeaders,
      payload
    );
  }
  listCollections(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        queries: rest[0],
        search: rest[1],
        total: rest[2]
      };
    }
    const databaseId = params.databaseId;
    const queries = params.queries;
    const search = params.search;
    const total = params.total;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    const apiPath = "/databases/{databaseId}/collections".replace("{databaseId}", databaseId);
    const payload = {};
    if (typeof queries !== "undefined") {
      payload["queries"] = queries;
    }
    if (typeof search !== "undefined") {
      payload["search"] = search;
    }
    if (typeof total !== "undefined") {
      payload["total"] = total;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {};
    return this.client.call(
      "get",
      uri,
      apiHeaders,
      payload
    );
  }
  createCollection(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        name: rest[1],
        permissions: rest[2],
        documentSecurity: rest[3],
        enabled: rest[4],
        attributes: rest[5],
        indexes: rest[6]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const name = params.name;
    const permissions = params.permissions;
    const documentSecurity = params.documentSecurity;
    const enabled = params.enabled;
    const attributes = params.attributes;
    const indexes = params.indexes;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof name === "undefined") {
      throw new AppwriteException('Missing required parameter: "name"');
    }
    const apiPath = "/databases/{databaseId}/collections".replace("{databaseId}", databaseId);
    const payload = {};
    if (typeof collectionId !== "undefined") {
      payload["collectionId"] = collectionId;
    }
    if (typeof name !== "undefined") {
      payload["name"] = name;
    }
    if (typeof permissions !== "undefined") {
      payload["permissions"] = permissions;
    }
    if (typeof documentSecurity !== "undefined") {
      payload["documentSecurity"] = documentSecurity;
    }
    if (typeof enabled !== "undefined") {
      payload["enabled"] = enabled;
    }
    if (typeof attributes !== "undefined") {
      payload["attributes"] = attributes;
    }
    if (typeof indexes !== "undefined") {
      payload["indexes"] = indexes;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "post",
      uri,
      apiHeaders,
      payload
    );
  }
  getCollection(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId);
    const payload = {};
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {};
    return this.client.call(
      "get",
      uri,
      apiHeaders,
      payload
    );
  }
  updateCollection(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        name: rest[1],
        permissions: rest[2],
        documentSecurity: rest[3],
        enabled: rest[4]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const name = params.name;
    const permissions = params.permissions;
    const documentSecurity = params.documentSecurity;
    const enabled = params.enabled;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof name === "undefined") {
      throw new AppwriteException('Missing required parameter: "name"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId);
    const payload = {};
    if (typeof name !== "undefined") {
      payload["name"] = name;
    }
    if (typeof permissions !== "undefined") {
      payload["permissions"] = permissions;
    }
    if (typeof documentSecurity !== "undefined") {
      payload["documentSecurity"] = documentSecurity;
    }
    if (typeof enabled !== "undefined") {
      payload["enabled"] = enabled;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "put",
      uri,
      apiHeaders,
      payload
    );
  }
  deleteCollection(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId);
    const payload = {};
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "delete",
      uri,
      apiHeaders,
      payload
    );
  }
  listAttributes(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        queries: rest[1],
        total: rest[2]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const queries = params.queries;
    const total = params.total;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId);
    const payload = {};
    if (typeof queries !== "undefined") {
      payload["queries"] = queries;
    }
    if (typeof total !== "undefined") {
      payload["total"] = total;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {};
    return this.client.call(
      "get",
      uri,
      apiHeaders,
      payload
    );
  }
  createBooleanAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        required: rest[2],
        xdefault: rest[3],
        array: rest[4]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const required = params.required;
    const xdefault = params.xdefault;
    const array = params.array;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    if (typeof required === "undefined") {
      throw new AppwriteException('Missing required parameter: "required"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/boolean".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId);
    const payload = {};
    if (typeof key !== "undefined") {
      payload["key"] = key;
    }
    if (typeof required !== "undefined") {
      payload["required"] = required;
    }
    if (typeof xdefault !== "undefined") {
      payload["default"] = xdefault;
    }
    if (typeof array !== "undefined") {
      payload["array"] = array;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "post",
      uri,
      apiHeaders,
      payload
    );
  }
  updateBooleanAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        required: rest[2],
        xdefault: rest[3],
        newKey: rest[4]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const required = params.required;
    const xdefault = params.xdefault;
    const newKey = params.newKey;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    if (typeof required === "undefined") {
      throw new AppwriteException('Missing required parameter: "required"');
    }
    if (typeof xdefault === "undefined") {
      throw new AppwriteException('Missing required parameter: "xdefault"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/boolean/{key}".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId).replace("{key}", key);
    const payload = {};
    if (typeof required !== "undefined") {
      payload["required"] = required;
    }
    if (typeof xdefault !== "undefined") {
      payload["default"] = xdefault;
    }
    if (typeof newKey !== "undefined") {
      payload["newKey"] = newKey;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "patch",
      uri,
      apiHeaders,
      payload
    );
  }
  createDatetimeAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        required: rest[2],
        xdefault: rest[3],
        array: rest[4]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const required = params.required;
    const xdefault = params.xdefault;
    const array = params.array;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    if (typeof required === "undefined") {
      throw new AppwriteException('Missing required parameter: "required"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/datetime".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId);
    const payload = {};
    if (typeof key !== "undefined") {
      payload["key"] = key;
    }
    if (typeof required !== "undefined") {
      payload["required"] = required;
    }
    if (typeof xdefault !== "undefined") {
      payload["default"] = xdefault;
    }
    if (typeof array !== "undefined") {
      payload["array"] = array;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "post",
      uri,
      apiHeaders,
      payload
    );
  }
  updateDatetimeAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        required: rest[2],
        xdefault: rest[3],
        newKey: rest[4]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const required = params.required;
    const xdefault = params.xdefault;
    const newKey = params.newKey;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    if (typeof required === "undefined") {
      throw new AppwriteException('Missing required parameter: "required"');
    }
    if (typeof xdefault === "undefined") {
      throw new AppwriteException('Missing required parameter: "xdefault"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/datetime/{key}".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId).replace("{key}", key);
    const payload = {};
    if (typeof required !== "undefined") {
      payload["required"] = required;
    }
    if (typeof xdefault !== "undefined") {
      payload["default"] = xdefault;
    }
    if (typeof newKey !== "undefined") {
      payload["newKey"] = newKey;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "patch",
      uri,
      apiHeaders,
      payload
    );
  }
  createEmailAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        required: rest[2],
        xdefault: rest[3],
        array: rest[4]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const required = params.required;
    const xdefault = params.xdefault;
    const array = params.array;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    if (typeof required === "undefined") {
      throw new AppwriteException('Missing required parameter: "required"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/email".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId);
    const payload = {};
    if (typeof key !== "undefined") {
      payload["key"] = key;
    }
    if (typeof required !== "undefined") {
      payload["required"] = required;
    }
    if (typeof xdefault !== "undefined") {
      payload["default"] = xdefault;
    }
    if (typeof array !== "undefined") {
      payload["array"] = array;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "post",
      uri,
      apiHeaders,
      payload
    );
  }
  updateEmailAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        required: rest[2],
        xdefault: rest[3],
        newKey: rest[4]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const required = params.required;
    const xdefault = params.xdefault;
    const newKey = params.newKey;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    if (typeof required === "undefined") {
      throw new AppwriteException('Missing required parameter: "required"');
    }
    if (typeof xdefault === "undefined") {
      throw new AppwriteException('Missing required parameter: "xdefault"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/email/{key}".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId).replace("{key}", key);
    const payload = {};
    if (typeof required !== "undefined") {
      payload["required"] = required;
    }
    if (typeof xdefault !== "undefined") {
      payload["default"] = xdefault;
    }
    if (typeof newKey !== "undefined") {
      payload["newKey"] = newKey;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "patch",
      uri,
      apiHeaders,
      payload
    );
  }
  createEnumAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        elements: rest[2],
        required: rest[3],
        xdefault: rest[4],
        array: rest[5]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const elements = params.elements;
    const required = params.required;
    const xdefault = params.xdefault;
    const array = params.array;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    if (typeof elements === "undefined") {
      throw new AppwriteException('Missing required parameter: "elements"');
    }
    if (typeof required === "undefined") {
      throw new AppwriteException('Missing required parameter: "required"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/enum".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId);
    const payload = {};
    if (typeof key !== "undefined") {
      payload["key"] = key;
    }
    if (typeof elements !== "undefined") {
      payload["elements"] = elements;
    }
    if (typeof required !== "undefined") {
      payload["required"] = required;
    }
    if (typeof xdefault !== "undefined") {
      payload["default"] = xdefault;
    }
    if (typeof array !== "undefined") {
      payload["array"] = array;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "post",
      uri,
      apiHeaders,
      payload
    );
  }
  updateEnumAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        elements: rest[2],
        required: rest[3],
        xdefault: rest[4],
        newKey: rest[5]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const elements = params.elements;
    const required = params.required;
    const xdefault = params.xdefault;
    const newKey = params.newKey;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    if (typeof elements === "undefined") {
      throw new AppwriteException('Missing required parameter: "elements"');
    }
    if (typeof required === "undefined") {
      throw new AppwriteException('Missing required parameter: "required"');
    }
    if (typeof xdefault === "undefined") {
      throw new AppwriteException('Missing required parameter: "xdefault"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/enum/{key}".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId).replace("{key}", key);
    const payload = {};
    if (typeof elements !== "undefined") {
      payload["elements"] = elements;
    }
    if (typeof required !== "undefined") {
      payload["required"] = required;
    }
    if (typeof xdefault !== "undefined") {
      payload["default"] = xdefault;
    }
    if (typeof newKey !== "undefined") {
      payload["newKey"] = newKey;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "patch",
      uri,
      apiHeaders,
      payload
    );
  }
  createFloatAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        required: rest[2],
        min: rest[3],
        max: rest[4],
        xdefault: rest[5],
        array: rest[6]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const required = params.required;
    const min = params.min;
    const max = params.max;
    const xdefault = params.xdefault;
    const array = params.array;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    if (typeof required === "undefined") {
      throw new AppwriteException('Missing required parameter: "required"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/float".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId);
    const payload = {};
    if (typeof key !== "undefined") {
      payload["key"] = key;
    }
    if (typeof required !== "undefined") {
      payload["required"] = required;
    }
    if (typeof min !== "undefined") {
      payload["min"] = min;
    }
    if (typeof max !== "undefined") {
      payload["max"] = max;
    }
    if (typeof xdefault !== "undefined") {
      payload["default"] = xdefault;
    }
    if (typeof array !== "undefined") {
      payload["array"] = array;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "post",
      uri,
      apiHeaders,
      payload
    );
  }
  updateFloatAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        required: rest[2],
        xdefault: rest[3],
        min: rest[4],
        max: rest[5],
        newKey: rest[6]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const required = params.required;
    const xdefault = params.xdefault;
    const min = params.min;
    const max = params.max;
    const newKey = params.newKey;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    if (typeof required === "undefined") {
      throw new AppwriteException('Missing required parameter: "required"');
    }
    if (typeof xdefault === "undefined") {
      throw new AppwriteException('Missing required parameter: "xdefault"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/float/{key}".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId).replace("{key}", key);
    const payload = {};
    if (typeof required !== "undefined") {
      payload["required"] = required;
    }
    if (typeof min !== "undefined") {
      payload["min"] = min;
    }
    if (typeof max !== "undefined") {
      payload["max"] = max;
    }
    if (typeof xdefault !== "undefined") {
      payload["default"] = xdefault;
    }
    if (typeof newKey !== "undefined") {
      payload["newKey"] = newKey;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "patch",
      uri,
      apiHeaders,
      payload
    );
  }
  createIntegerAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        required: rest[2],
        min: rest[3],
        max: rest[4],
        xdefault: rest[5],
        array: rest[6]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const required = params.required;
    const min = params.min;
    const max = params.max;
    const xdefault = params.xdefault;
    const array = params.array;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    if (typeof required === "undefined") {
      throw new AppwriteException('Missing required parameter: "required"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/integer".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId);
    const payload = {};
    if (typeof key !== "undefined") {
      payload["key"] = key;
    }
    if (typeof required !== "undefined") {
      payload["required"] = required;
    }
    if (typeof min !== "undefined") {
      payload["min"] = min;
    }
    if (typeof max !== "undefined") {
      payload["max"] = max;
    }
    if (typeof xdefault !== "undefined") {
      payload["default"] = xdefault;
    }
    if (typeof array !== "undefined") {
      payload["array"] = array;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "post",
      uri,
      apiHeaders,
      payload
    );
  }
  updateIntegerAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        required: rest[2],
        xdefault: rest[3],
        min: rest[4],
        max: rest[5],
        newKey: rest[6]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const required = params.required;
    const xdefault = params.xdefault;
    const min = params.min;
    const max = params.max;
    const newKey = params.newKey;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    if (typeof required === "undefined") {
      throw new AppwriteException('Missing required parameter: "required"');
    }
    if (typeof xdefault === "undefined") {
      throw new AppwriteException('Missing required parameter: "xdefault"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/integer/{key}".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId).replace("{key}", key);
    const payload = {};
    if (typeof required !== "undefined") {
      payload["required"] = required;
    }
    if (typeof min !== "undefined") {
      payload["min"] = min;
    }
    if (typeof max !== "undefined") {
      payload["max"] = max;
    }
    if (typeof xdefault !== "undefined") {
      payload["default"] = xdefault;
    }
    if (typeof newKey !== "undefined") {
      payload["newKey"] = newKey;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "patch",
      uri,
      apiHeaders,
      payload
    );
  }
  createIpAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        required: rest[2],
        xdefault: rest[3],
        array: rest[4]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const required = params.required;
    const xdefault = params.xdefault;
    const array = params.array;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    if (typeof required === "undefined") {
      throw new AppwriteException('Missing required parameter: "required"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/ip".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId);
    const payload = {};
    if (typeof key !== "undefined") {
      payload["key"] = key;
    }
    if (typeof required !== "undefined") {
      payload["required"] = required;
    }
    if (typeof xdefault !== "undefined") {
      payload["default"] = xdefault;
    }
    if (typeof array !== "undefined") {
      payload["array"] = array;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "post",
      uri,
      apiHeaders,
      payload
    );
  }
  updateIpAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        required: rest[2],
        xdefault: rest[3],
        newKey: rest[4]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const required = params.required;
    const xdefault = params.xdefault;
    const newKey = params.newKey;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    if (typeof required === "undefined") {
      throw new AppwriteException('Missing required parameter: "required"');
    }
    if (typeof xdefault === "undefined") {
      throw new AppwriteException('Missing required parameter: "xdefault"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/ip/{key}".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId).replace("{key}", key);
    const payload = {};
    if (typeof required !== "undefined") {
      payload["required"] = required;
    }
    if (typeof xdefault !== "undefined") {
      payload["default"] = xdefault;
    }
    if (typeof newKey !== "undefined") {
      payload["newKey"] = newKey;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "patch",
      uri,
      apiHeaders,
      payload
    );
  }
  createLineAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        required: rest[2],
        xdefault: rest[3]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const required = params.required;
    const xdefault = params.xdefault;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    if (typeof required === "undefined") {
      throw new AppwriteException('Missing required parameter: "required"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/line".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId);
    const payload = {};
    if (typeof key !== "undefined") {
      payload["key"] = key;
    }
    if (typeof required !== "undefined") {
      payload["required"] = required;
    }
    if (typeof xdefault !== "undefined") {
      payload["default"] = xdefault;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "post",
      uri,
      apiHeaders,
      payload
    );
  }
  updateLineAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        required: rest[2],
        xdefault: rest[3],
        newKey: rest[4]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const required = params.required;
    const xdefault = params.xdefault;
    const newKey = params.newKey;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    if (typeof required === "undefined") {
      throw new AppwriteException('Missing required parameter: "required"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/line/{key}".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId).replace("{key}", key);
    const payload = {};
    if (typeof required !== "undefined") {
      payload["required"] = required;
    }
    if (typeof xdefault !== "undefined") {
      payload["default"] = xdefault;
    }
    if (typeof newKey !== "undefined") {
      payload["newKey"] = newKey;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "patch",
      uri,
      apiHeaders,
      payload
    );
  }
  createPointAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        required: rest[2],
        xdefault: rest[3]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const required = params.required;
    const xdefault = params.xdefault;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    if (typeof required === "undefined") {
      throw new AppwriteException('Missing required parameter: "required"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/point".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId);
    const payload = {};
    if (typeof key !== "undefined") {
      payload["key"] = key;
    }
    if (typeof required !== "undefined") {
      payload["required"] = required;
    }
    if (typeof xdefault !== "undefined") {
      payload["default"] = xdefault;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "post",
      uri,
      apiHeaders,
      payload
    );
  }
  updatePointAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        required: rest[2],
        xdefault: rest[3],
        newKey: rest[4]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const required = params.required;
    const xdefault = params.xdefault;
    const newKey = params.newKey;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    if (typeof required === "undefined") {
      throw new AppwriteException('Missing required parameter: "required"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/point/{key}".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId).replace("{key}", key);
    const payload = {};
    if (typeof required !== "undefined") {
      payload["required"] = required;
    }
    if (typeof xdefault !== "undefined") {
      payload["default"] = xdefault;
    }
    if (typeof newKey !== "undefined") {
      payload["newKey"] = newKey;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "patch",
      uri,
      apiHeaders,
      payload
    );
  }
  createPolygonAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        required: rest[2],
        xdefault: rest[3]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const required = params.required;
    const xdefault = params.xdefault;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    if (typeof required === "undefined") {
      throw new AppwriteException('Missing required parameter: "required"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/polygon".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId);
    const payload = {};
    if (typeof key !== "undefined") {
      payload["key"] = key;
    }
    if (typeof required !== "undefined") {
      payload["required"] = required;
    }
    if (typeof xdefault !== "undefined") {
      payload["default"] = xdefault;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "post",
      uri,
      apiHeaders,
      payload
    );
  }
  updatePolygonAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        required: rest[2],
        xdefault: rest[3],
        newKey: rest[4]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const required = params.required;
    const xdefault = params.xdefault;
    const newKey = params.newKey;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    if (typeof required === "undefined") {
      throw new AppwriteException('Missing required parameter: "required"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/polygon/{key}".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId).replace("{key}", key);
    const payload = {};
    if (typeof required !== "undefined") {
      payload["required"] = required;
    }
    if (typeof xdefault !== "undefined") {
      payload["default"] = xdefault;
    }
    if (typeof newKey !== "undefined") {
      payload["newKey"] = newKey;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "patch",
      uri,
      apiHeaders,
      payload
    );
  }
  createRelationshipAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        relatedCollectionId: rest[1],
        type: rest[2],
        twoWay: rest[3],
        key: rest[4],
        twoWayKey: rest[5],
        onDelete: rest[6]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const relatedCollectionId = params.relatedCollectionId;
    const type = params.type;
    const twoWay = params.twoWay;
    const key = params.key;
    const twoWayKey = params.twoWayKey;
    const onDelete = params.onDelete;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof relatedCollectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "relatedCollectionId"');
    }
    if (typeof type === "undefined") {
      throw new AppwriteException('Missing required parameter: "type"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/relationship".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId);
    const payload = {};
    if (typeof relatedCollectionId !== "undefined") {
      payload["relatedCollectionId"] = relatedCollectionId;
    }
    if (typeof type !== "undefined") {
      payload["type"] = type;
    }
    if (typeof twoWay !== "undefined") {
      payload["twoWay"] = twoWay;
    }
    if (typeof key !== "undefined") {
      payload["key"] = key;
    }
    if (typeof twoWayKey !== "undefined") {
      payload["twoWayKey"] = twoWayKey;
    }
    if (typeof onDelete !== "undefined") {
      payload["onDelete"] = onDelete;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "post",
      uri,
      apiHeaders,
      payload
    );
  }
  createStringAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        size: rest[2],
        required: rest[3],
        xdefault: rest[4],
        array: rest[5],
        encrypt: rest[6]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const size = params.size;
    const required = params.required;
    const xdefault = params.xdefault;
    const array = params.array;
    const encrypt = params.encrypt;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    if (typeof size === "undefined") {
      throw new AppwriteException('Missing required parameter: "size"');
    }
    if (typeof required === "undefined") {
      throw new AppwriteException('Missing required parameter: "required"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/string".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId);
    const payload = {};
    if (typeof key !== "undefined") {
      payload["key"] = key;
    }
    if (typeof size !== "undefined") {
      payload["size"] = size;
    }
    if (typeof required !== "undefined") {
      payload["required"] = required;
    }
    if (typeof xdefault !== "undefined") {
      payload["default"] = xdefault;
    }
    if (typeof array !== "undefined") {
      payload["array"] = array;
    }
    if (typeof encrypt !== "undefined") {
      payload["encrypt"] = encrypt;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "post",
      uri,
      apiHeaders,
      payload
    );
  }
  updateStringAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        required: rest[2],
        xdefault: rest[3],
        size: rest[4],
        newKey: rest[5]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const required = params.required;
    const xdefault = params.xdefault;
    const size = params.size;
    const newKey = params.newKey;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    if (typeof required === "undefined") {
      throw new AppwriteException('Missing required parameter: "required"');
    }
    if (typeof xdefault === "undefined") {
      throw new AppwriteException('Missing required parameter: "xdefault"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/string/{key}".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId).replace("{key}", key);
    const payload = {};
    if (typeof required !== "undefined") {
      payload["required"] = required;
    }
    if (typeof xdefault !== "undefined") {
      payload["default"] = xdefault;
    }
    if (typeof size !== "undefined") {
      payload["size"] = size;
    }
    if (typeof newKey !== "undefined") {
      payload["newKey"] = newKey;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "patch",
      uri,
      apiHeaders,
      payload
    );
  }
  createUrlAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        required: rest[2],
        xdefault: rest[3],
        array: rest[4]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const required = params.required;
    const xdefault = params.xdefault;
    const array = params.array;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    if (typeof required === "undefined") {
      throw new AppwriteException('Missing required parameter: "required"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/url".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId);
    const payload = {};
    if (typeof key !== "undefined") {
      payload["key"] = key;
    }
    if (typeof required !== "undefined") {
      payload["required"] = required;
    }
    if (typeof xdefault !== "undefined") {
      payload["default"] = xdefault;
    }
    if (typeof array !== "undefined") {
      payload["array"] = array;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "post",
      uri,
      apiHeaders,
      payload
    );
  }
  updateUrlAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        required: rest[2],
        xdefault: rest[3],
        newKey: rest[4]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const required = params.required;
    const xdefault = params.xdefault;
    const newKey = params.newKey;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    if (typeof required === "undefined") {
      throw new AppwriteException('Missing required parameter: "required"');
    }
    if (typeof xdefault === "undefined") {
      throw new AppwriteException('Missing required parameter: "xdefault"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/url/{key}".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId).replace("{key}", key);
    const payload = {};
    if (typeof required !== "undefined") {
      payload["required"] = required;
    }
    if (typeof xdefault !== "undefined") {
      payload["default"] = xdefault;
    }
    if (typeof newKey !== "undefined") {
      payload["newKey"] = newKey;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "patch",
      uri,
      apiHeaders,
      payload
    );
  }
  getAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/{key}".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId).replace("{key}", key);
    const payload = {};
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {};
    return this.client.call(
      "get",
      uri,
      apiHeaders,
      payload
    );
  }
  deleteAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/{key}".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId).replace("{key}", key);
    const payload = {};
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "delete",
      uri,
      apiHeaders,
      payload
    );
  }
  updateRelationshipAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        onDelete: rest[2],
        newKey: rest[3]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const onDelete = params.onDelete;
    const newKey = params.newKey;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/attributes/{key}/relationship".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId).replace("{key}", key);
    const payload = {};
    if (typeof onDelete !== "undefined") {
      payload["onDelete"] = onDelete;
    }
    if (typeof newKey !== "undefined") {
      payload["newKey"] = newKey;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "patch",
      uri,
      apiHeaders,
      payload
    );
  }
  listDocuments(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        queries: rest[1],
        transactionId: rest[2],
        total: rest[3]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const queries = params.queries;
    const transactionId = params.transactionId;
    const total = params.total;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/documents".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId);
    const payload = {};
    if (typeof queries !== "undefined") {
      payload["queries"] = queries;
    }
    if (typeof transactionId !== "undefined") {
      payload["transactionId"] = transactionId;
    }
    if (typeof total !== "undefined") {
      payload["total"] = total;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {};
    return this.client.call(
      "get",
      uri,
      apiHeaders,
      payload
    );
  }
  createDocument(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        documentId: rest[1],
        data: rest[2],
        permissions: rest[3],
        transactionId: rest[4]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const documentId = params.documentId;
    const data = params.data;
    const permissions = params.permissions;
    const transactionId = params.transactionId;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof documentId === "undefined") {
      throw new AppwriteException('Missing required parameter: "documentId"');
    }
    if (typeof data === "undefined") {
      throw new AppwriteException('Missing required parameter: "data"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/documents".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId);
    const payload = {};
    if (typeof documentId !== "undefined") {
      payload["documentId"] = documentId;
    }
    if (typeof data !== "undefined") {
      payload["data"] = data;
    }
    if (typeof permissions !== "undefined") {
      payload["permissions"] = permissions;
    }
    if (typeof transactionId !== "undefined") {
      payload["transactionId"] = transactionId;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "post",
      uri,
      apiHeaders,
      payload
    );
  }
  createDocuments(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        documents: rest[1],
        transactionId: rest[2]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const documents = params.documents;
    const transactionId = params.transactionId;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof documents === "undefined") {
      throw new AppwriteException('Missing required parameter: "documents"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/documents".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId);
    const payload = {};
    if (typeof documents !== "undefined") {
      payload["documents"] = documents;
    }
    if (typeof transactionId !== "undefined") {
      payload["transactionId"] = transactionId;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "post",
      uri,
      apiHeaders,
      payload
    );
  }
  upsertDocuments(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        documents: rest[1],
        transactionId: rest[2]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const documents = params.documents;
    const transactionId = params.transactionId;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof documents === "undefined") {
      throw new AppwriteException('Missing required parameter: "documents"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/documents".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId);
    const payload = {};
    if (typeof documents !== "undefined") {
      payload["documents"] = documents;
    }
    if (typeof transactionId !== "undefined") {
      payload["transactionId"] = transactionId;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "put",
      uri,
      apiHeaders,
      payload
    );
  }
  updateDocuments(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        data: rest[1],
        queries: rest[2],
        transactionId: rest[3]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const data = params.data;
    const queries = params.queries;
    const transactionId = params.transactionId;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/documents".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId);
    const payload = {};
    if (typeof data !== "undefined") {
      payload["data"] = data;
    }
    if (typeof queries !== "undefined") {
      payload["queries"] = queries;
    }
    if (typeof transactionId !== "undefined") {
      payload["transactionId"] = transactionId;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "patch",
      uri,
      apiHeaders,
      payload
    );
  }
  deleteDocuments(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        queries: rest[1],
        transactionId: rest[2]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const queries = params.queries;
    const transactionId = params.transactionId;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/documents".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId);
    const payload = {};
    if (typeof queries !== "undefined") {
      payload["queries"] = queries;
    }
    if (typeof transactionId !== "undefined") {
      payload["transactionId"] = transactionId;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "delete",
      uri,
      apiHeaders,
      payload
    );
  }
  getDocument(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        documentId: rest[1],
        queries: rest[2],
        transactionId: rest[3]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const documentId = params.documentId;
    const queries = params.queries;
    const transactionId = params.transactionId;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof documentId === "undefined") {
      throw new AppwriteException('Missing required parameter: "documentId"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/documents/{documentId}".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId).replace("{documentId}", documentId);
    const payload = {};
    if (typeof queries !== "undefined") {
      payload["queries"] = queries;
    }
    if (typeof transactionId !== "undefined") {
      payload["transactionId"] = transactionId;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {};
    return this.client.call(
      "get",
      uri,
      apiHeaders,
      payload
    );
  }
  upsertDocument(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        documentId: rest[1],
        data: rest[2],
        permissions: rest[3],
        transactionId: rest[4]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const documentId = params.documentId;
    const data = params.data;
    const permissions = params.permissions;
    const transactionId = params.transactionId;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof documentId === "undefined") {
      throw new AppwriteException('Missing required parameter: "documentId"');
    }
    if (typeof data === "undefined") {
      throw new AppwriteException('Missing required parameter: "data"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/documents/{documentId}".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId).replace("{documentId}", documentId);
    const payload = {};
    if (typeof data !== "undefined") {
      payload["data"] = data;
    }
    if (typeof permissions !== "undefined") {
      payload["permissions"] = permissions;
    }
    if (typeof transactionId !== "undefined") {
      payload["transactionId"] = transactionId;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "put",
      uri,
      apiHeaders,
      payload
    );
  }
  updateDocument(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        documentId: rest[1],
        data: rest[2],
        permissions: rest[3],
        transactionId: rest[4]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const documentId = params.documentId;
    const data = params.data;
    const permissions = params.permissions;
    const transactionId = params.transactionId;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof documentId === "undefined") {
      throw new AppwriteException('Missing required parameter: "documentId"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/documents/{documentId}".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId).replace("{documentId}", documentId);
    const payload = {};
    if (typeof data !== "undefined") {
      payload["data"] = data;
    }
    if (typeof permissions !== "undefined") {
      payload["permissions"] = permissions;
    }
    if (typeof transactionId !== "undefined") {
      payload["transactionId"] = transactionId;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "patch",
      uri,
      apiHeaders,
      payload
    );
  }
  deleteDocument(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        documentId: rest[1],
        transactionId: rest[2]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const documentId = params.documentId;
    const transactionId = params.transactionId;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof documentId === "undefined") {
      throw new AppwriteException('Missing required parameter: "documentId"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/documents/{documentId}".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId).replace("{documentId}", documentId);
    const payload = {};
    if (typeof transactionId !== "undefined") {
      payload["transactionId"] = transactionId;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "delete",
      uri,
      apiHeaders,
      payload
    );
  }
  decrementDocumentAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        documentId: rest[1],
        attribute: rest[2],
        value: rest[3],
        min: rest[4],
        transactionId: rest[5]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const documentId = params.documentId;
    const attribute = params.attribute;
    const value = params.value;
    const min = params.min;
    const transactionId = params.transactionId;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof documentId === "undefined") {
      throw new AppwriteException('Missing required parameter: "documentId"');
    }
    if (typeof attribute === "undefined") {
      throw new AppwriteException('Missing required parameter: "attribute"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/documents/{documentId}/{attribute}/decrement".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId).replace("{documentId}", documentId).replace("{attribute}", attribute);
    const payload = {};
    if (typeof value !== "undefined") {
      payload["value"] = value;
    }
    if (typeof min !== "undefined") {
      payload["min"] = min;
    }
    if (typeof transactionId !== "undefined") {
      payload["transactionId"] = transactionId;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "patch",
      uri,
      apiHeaders,
      payload
    );
  }
  incrementDocumentAttribute(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        documentId: rest[1],
        attribute: rest[2],
        value: rest[3],
        max: rest[4],
        transactionId: rest[5]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const documentId = params.documentId;
    const attribute = params.attribute;
    const value = params.value;
    const max = params.max;
    const transactionId = params.transactionId;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof documentId === "undefined") {
      throw new AppwriteException('Missing required parameter: "documentId"');
    }
    if (typeof attribute === "undefined") {
      throw new AppwriteException('Missing required parameter: "attribute"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/documents/{documentId}/{attribute}/increment".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId).replace("{documentId}", documentId).replace("{attribute}", attribute);
    const payload = {};
    if (typeof value !== "undefined") {
      payload["value"] = value;
    }
    if (typeof max !== "undefined") {
      payload["max"] = max;
    }
    if (typeof transactionId !== "undefined") {
      payload["transactionId"] = transactionId;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "patch",
      uri,
      apiHeaders,
      payload
    );
  }
  listIndexes(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        queries: rest[1],
        total: rest[2]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const queries = params.queries;
    const total = params.total;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/indexes".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId);
    const payload = {};
    if (typeof queries !== "undefined") {
      payload["queries"] = queries;
    }
    if (typeof total !== "undefined") {
      payload["total"] = total;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {};
    return this.client.call(
      "get",
      uri,
      apiHeaders,
      payload
    );
  }
  createIndex(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1],
        type: rest[2],
        attributes: rest[3],
        orders: rest[4],
        lengths: rest[5]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    const type = params.type;
    const attributes = params.attributes;
    const orders = params.orders;
    const lengths = params.lengths;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    if (typeof type === "undefined") {
      throw new AppwriteException('Missing required parameter: "type"');
    }
    if (typeof attributes === "undefined") {
      throw new AppwriteException('Missing required parameter: "attributes"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/indexes".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId);
    const payload = {};
    if (typeof key !== "undefined") {
      payload["key"] = key;
    }
    if (typeof type !== "undefined") {
      payload["type"] = type;
    }
    if (typeof attributes !== "undefined") {
      payload["attributes"] = attributes;
    }
    if (typeof orders !== "undefined") {
      payload["orders"] = orders;
    }
    if (typeof lengths !== "undefined") {
      payload["lengths"] = lengths;
    }
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "post",
      uri,
      apiHeaders,
      payload
    );
  }
  getIndex(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/indexes/{key}".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId).replace("{key}", key);
    const payload = {};
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {};
    return this.client.call(
      "get",
      uri,
      apiHeaders,
      payload
    );
  }
  deleteIndex(paramsOrFirst, ...rest) {
    let params;
    if (paramsOrFirst && typeof paramsOrFirst === "object" && !Array.isArray(paramsOrFirst)) {
      params = paramsOrFirst || {};
    } else {
      params = {
        databaseId: paramsOrFirst,
        collectionId: rest[0],
        key: rest[1]
      };
    }
    const databaseId = params.databaseId;
    const collectionId = params.collectionId;
    const key = params.key;
    if (typeof databaseId === "undefined") {
      throw new AppwriteException('Missing required parameter: "databaseId"');
    }
    if (typeof collectionId === "undefined") {
      throw new AppwriteException('Missing required parameter: "collectionId"');
    }
    if (typeof key === "undefined") {
      throw new AppwriteException('Missing required parameter: "key"');
    }
    const apiPath = "/databases/{databaseId}/collections/{collectionId}/indexes/{key}".replace("{databaseId}", databaseId).replace("{collectionId}", collectionId).replace("{key}", key);
    const payload = {};
    const uri = new URL(this.client.config.endpoint + apiPath);
    const apiHeaders = {
      "content-type": "application/json"
    };
    return this.client.call(
      "delete",
      uri,
      apiHeaders,
      payload
    );
  }
};

// node_modules/node-appwrite/dist/permission.mjs
var Permission = class {
  static {
    __name(this, "Permission");
  }
};
Permission.read = (role) => {
  return `read("${role}")`;
};
Permission.write = (role) => {
  return `write("${role}")`;
};
Permission.create = (role) => {
  return `create("${role}")`;
};
Permission.update = (role) => {
  return `update("${role}")`;
};
Permission.delete = (role) => {
  return `delete("${role}")`;
};

// node_modules/node-appwrite/dist/id.mjs
var ID = class _ID {
  static {
    __name(this, "_ID");
  }
  /**
   * Generate an hex ID based on timestamp.
   * Recreated from https://www.php.net/manual/en/function.uniqid.php
   *
   * @returns {string}
   */
  static #hexTimestamp() {
    const now = /* @__PURE__ */ new Date();
    const sec = Math.floor(now.getTime() / 1e3);
    const msec = now.getMilliseconds();
    const hexTimestamp = sec.toString(16) + msec.toString(16).padStart(5, "0");
    return hexTimestamp;
  }
  /**
   * Uses the provided ID as the ID for the resource.
   *
   * @param {string} id
   * @returns {string}
   */
  static custom(id) {
    return id;
  }
  /**
   * Have Appwrite generate a unique ID for you.
   * 
   * @param {number} padding. Default is 7.
   * @returns {string}
   */
  static unique(padding = 7) {
    const baseId = _ID.#hexTimestamp();
    let randomPadding = "";
    for (let i3 = 0; i3 < padding; i3++) {
      const randomHexDigit = Math.floor(Math.random() * 16).toString(16);
      randomPadding += randomHexDigit;
    }
    return baseId + randomPadding;
  }
};

// node_modules/node-appwrite/dist/operator.mjs
var _Operator = class _Operator2 {
  static {
    __name(this, "_Operator");
  }
  /**
   * Constructor for Operator class.
   *
   * @param {string} method
   * @param {OperatorValues} values
   */
  constructor(method, values) {
    this.method = method;
    if (values !== void 0) {
      if (Array.isArray(values)) {
        this.values = values;
      } else {
        this.values = [values];
      }
    }
  }
  /**
   * Convert the operator object to a JSON string.
   *
   * @returns {string}
   */
  toString() {
    return JSON.stringify({
      method: this.method,
      values: this.values
    });
  }
};
_Operator.increment = (value = 1, max) => {
  if (isNaN(value) || !isFinite(value)) {
    throw new Error("Value cannot be NaN or Infinity");
  }
  if (max !== void 0 && (isNaN(max) || !isFinite(max))) {
    throw new Error("Max cannot be NaN or Infinity");
  }
  const values = [value];
  if (max !== void 0) {
    values.push(max);
  }
  return new _Operator("increment", values).toString();
};
_Operator.decrement = (value = 1, min) => {
  if (isNaN(value) || !isFinite(value)) {
    throw new Error("Value cannot be NaN or Infinity");
  }
  if (min !== void 0 && (isNaN(min) || !isFinite(min))) {
    throw new Error("Min cannot be NaN or Infinity");
  }
  const values = [value];
  if (min !== void 0) {
    values.push(min);
  }
  return new _Operator("decrement", values).toString();
};
_Operator.multiply = (factor, max) => {
  if (isNaN(factor) || !isFinite(factor)) {
    throw new Error("Factor cannot be NaN or Infinity");
  }
  if (max !== void 0 && (isNaN(max) || !isFinite(max))) {
    throw new Error("Max cannot be NaN or Infinity");
  }
  const values = [factor];
  if (max !== void 0) {
    values.push(max);
  }
  return new _Operator("multiply", values).toString();
};
_Operator.divide = (divisor, min) => {
  if (isNaN(divisor) || !isFinite(divisor)) {
    throw new Error("Divisor cannot be NaN or Infinity");
  }
  if (min !== void 0 && (isNaN(min) || !isFinite(min))) {
    throw new Error("Min cannot be NaN or Infinity");
  }
  if (divisor === 0) {
    throw new Error("Divisor cannot be zero");
  }
  const values = [divisor];
  if (min !== void 0) {
    values.push(min);
  }
  return new _Operator("divide", values).toString();
};
_Operator.modulo = (divisor) => {
  if (isNaN(divisor) || !isFinite(divisor)) {
    throw new Error("Divisor cannot be NaN or Infinity");
  }
  if (divisor === 0) {
    throw new Error("Divisor cannot be zero");
  }
  return new _Operator("modulo", [divisor]).toString();
};
_Operator.power = (exponent, max) => {
  if (isNaN(exponent) || !isFinite(exponent)) {
    throw new Error("Exponent cannot be NaN or Infinity");
  }
  if (max !== void 0 && (isNaN(max) || !isFinite(max))) {
    throw new Error("Max cannot be NaN or Infinity");
  }
  const values = [exponent];
  if (max !== void 0) {
    values.push(max);
  }
  return new _Operator("power", values).toString();
};
_Operator.arrayAppend = (values) => new _Operator("arrayAppend", values).toString();
_Operator.arrayPrepend = (values) => new _Operator("arrayPrepend", values).toString();
_Operator.arrayInsert = (index, value) => new _Operator("arrayInsert", [index, value]).toString();
_Operator.arrayRemove = (value) => new _Operator("arrayRemove", [value]).toString();
_Operator.arrayUnique = () => new _Operator("arrayUnique", []).toString();
_Operator.arrayIntersect = (values) => new _Operator("arrayIntersect", values).toString();
_Operator.arrayDiff = (values) => new _Operator("arrayDiff", values).toString();
_Operator.arrayFilter = (condition, value) => {
  const values = [condition, value === void 0 ? null : value];
  return new _Operator("arrayFilter", values).toString();
};
_Operator.stringConcat = (value) => new _Operator("stringConcat", [value]).toString();
_Operator.stringReplace = (search, replace) => new _Operator("stringReplace", [search, replace]).toString();
_Operator.toggle = () => new _Operator("toggle", []).toString();
_Operator.dateAddDays = (days) => new _Operator("dateAddDays", [days]).toString();
_Operator.dateSubDays = (days) => new _Operator("dateSubDays", [days]).toString();
_Operator.dateSetNow = () => new _Operator("dateSetNow", []).toString();

// src/lib/appwrite.ts
var AppwriteService = class {
  static {
    __name(this, "AppwriteService");
  }
  constructor(env2) {
    this.env = env2;
    this.client = new Client().setEndpoint(env2.APPWRITE_ENDPOINT).setProject(env2.APPWRITE_PROJECT_ID).setKey(env2.APPWRITE_API_KEY);
    this.databases = new Databases(this.client);
  }
  async createTicket(ticketId, amount) {
    try {
      return await this.databases.createDocument(
        this.env.APPWRITE_DATABASE_ID,
        this.env.APPWRITE_COLLECTION_ID,
        ID.unique(),
        // Appwrite ID
        {
          ticketId,
          amount,
          status: "pending"
        }
      );
    } catch (error3) {
      console.error("Appwrite createTicket error:", error3);
      throw error3;
    }
  }
  async markAsPaid(ticketId) {
    try {
      const response = await this.databases.listDocuments(
        this.env.APPWRITE_DATABASE_ID,
        this.env.APPWRITE_COLLECTION_ID,
        [Query.equal("ticketId", ticketId)]
      );
      if (response.documents.length === 0) {
        return false;
      }
      const docId = response.documents[0].$id;
      await this.databases.updateDocument(
        this.env.APPWRITE_DATABASE_ID,
        this.env.APPWRITE_COLLECTION_ID,
        docId,
        {
          status: "paid"
        }
      );
      return true;
    } catch (error3) {
      console.error("Appwrite markAsPaid error:", error3);
      return false;
    }
  }
  async getTicketStatus(ticketId) {
    try {
      const response = await this.databases.listDocuments(
        this.env.APPWRITE_DATABASE_ID,
        this.env.APPWRITE_COLLECTION_ID,
        [Query.equal("ticketId", ticketId)]
      );
      if (response.documents.length === 0) {
        return null;
      }
      const doc = response.documents[0];
      return {
        ticketId: doc.ticketId,
        status: doc.status,
        amount: doc.amount
      };
    } catch (error3) {
      console.error("Appwrite getTicketStatus error:", error3);
      return null;
    }
  }
};

// src/worker.ts
var app = new Hono2();
app.use("/*", cors());
app.get("/", (c) => {
  return c.text("Payment Gateway API is running!");
});
app.post("/api/ticket", async (c) => {
  try {
    const body = await c.req.json();
    const amount = body.amount;
    if (!amount || typeof amount !== "number") {
      return c.json({ error: "Invalid amount" }, 400);
    }
    const timestamp = Date.now();
    const ticketId = `TICKET${timestamp}`;
    const appwrite = new AppwriteService(c.env);
    await appwrite.createTicket(ticketId, amount);
    return c.json({
      id: ticketId,
      amount,
      status: "pending"
    });
  } catch (error3) {
    console.error("Error creating ticket:", error3);
    return c.json({ error: "Internal Server Error" }, 500);
  }
});
app.post("/api/webhook", async (c) => {
  try {
    const rawBody = await c.req.text();
    console.log("--- WEBHOOK RECEIVED ---");
    console.log("TIMESTAMP:", (/* @__PURE__ */ new Date()).toISOString());
    console.log("RAW BODY:", rawBody);
    let body;
    try {
      body = JSON.parse(rawBody);
    } catch (e3) {
      console.log("Could not parse JSON body, assuming raw text or URL encoded.");
      body = { sms: rawBody };
    }
    const secret = c.req.query("secret") || body.secret_key;
    if (secret !== c.env.WEBHOOK_SECRET) {
      console.log("Unauthorized Webhook Attempt");
      return c.json({ error: "Unauthorized" }, 401);
    }
    const smsText = body.sms || body.message || rawBody;
    const ticketMatch = smsText.match(/TICKET(\d+)/);
    let foundId = null;
    let status = "ignored";
    let updated = false;
    if (ticketMatch) {
      foundId = ticketMatch[0];
      console.log("FOUND TICKET ID:", foundId);
      const appwrite = new AppwriteService(c.env);
      updated = await appwrite.markAsPaid(foundId);
      if (updated) {
        console.log(`Ticket ${foundId} MARKED AS PAID`);
        status = "success";
      } else {
        console.log(`Ticket ${foundId} NOT FOUND or already paid`);
        status = "not_found_or_error";
      }
    } else {
      console.log("NO TICKET ID FOUND IN SMS");
    }
    return c.json({
      status: "received",
      processed_id: foundId,
      action: status
    });
  } catch (error3) {
    console.error("Webhook error:", error3);
    return c.json({ error: "Internal Server Error" }, 500);
  }
});
app.get("/api/status/:id", async (c) => {
  const id = c.req.param("id");
  const appwrite = new AppwriteService(c.env);
  const status = await appwrite.getTicketStatus(id);
  if (!status) {
    return c.json({ status: "not_found" }, 404);
  }
  return c.json(status);
});
var worker_default = app;
export {
  worker_default as default
};
//# sourceMappingURL=worker.js.map
