export function hasDuplicate<T>(items: T[]): boolean {
  const s = new Set(items);
  return Array.from(s.values()).length < items.length;
}
