interface TelemetryEvent {
  type: string;
  [key: string]: unknown;
}

export class TelemetryEmitter {
  private static sink: TelemetrySink = console;

  static setSink(sink: TelemetrySink) {
    this.sink = sink;
  }

  static emitPublishEvent(event: TelemetryEvent) {
    this.dispatch({ ...event, emittedAt: new Date().toISOString() });
  }

  static emitOfflinePackaging(event: TelemetryEvent) {
    this.dispatch({ ...event, emittedAt: new Date().toISOString() });
  }

  private static dispatch(event: TelemetryEvent) {
    if (this.sink && typeof this.sink.log === "function") {
      this.sink.log(`[telemetry] ${event.type}`, JSON.stringify(event));
    }
  }
}

export interface TelemetrySink {
  log(message?: any, ...optionalParams: any[]): void;
}
