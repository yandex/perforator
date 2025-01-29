export type LogLevel =
| 'info'
| 'debug'
| 'warn'
| 'error'
| 'critical'
| 'trace';

export type SendError = (error: unknown, additional?: Record<string, any>, level?: LogLevel) => void;
