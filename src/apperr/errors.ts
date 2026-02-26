export class AppError extends Error {
  constructor(
    message: string,
    public readonly kind: ErrorKind,
    options?: ErrorOptions,
  ) {
    super(message, options);
  }
}

export type ErrorKind = "usage" | "preflight" | "auth" | "cleanup" | "silent";

export function isAppError(err: unknown, kind: ErrorKind): boolean {
  if (err instanceof AppError) {
    return err.kind === kind;
  }
  if (err instanceof Error && err.cause) {
    return isAppError(err.cause, kind);
  }
  return false;
}

export function usageError(msg: string): AppError {
  return new AppError(msg, "usage");
}

export function preflightError(msg: string): AppError {
  return new AppError(msg, "preflight");
}

export function authError(msg: string): AppError {
  return new AppError(msg, "auth");
}

export function cleanupError(msg: string, cause?: Error): AppError {
  return new AppError(msg, "cleanup", { cause });
}

export function silentError(): AppError {
  return new AppError("", "silent");
}
