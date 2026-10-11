// Приводит неизвестную ошибку бэкенда к тексту, пригодному для показа пользователю.
export function toMessage(err: unknown): string {
  if (err instanceof Error) {
    return err.message;
  }
  return String(err);
}
