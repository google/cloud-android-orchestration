export function hasDuplicate<T>(items: T[]): boolean {
  const s = new Set(items);
  return Array.from(s.values()).length < items.length;
}

export function handleUrl(url: string | null | undefined): string {
  if (!url) {
    return '';
  }

  if (!url.endsWith('/')) {
    return url;
  }

  return url.slice(0, url.length - 1);
}

export function adjustArrayLength<T>(arr: T[], length: number, placeholder: T) {
  const currentArrayLength = arr.length;

  for (let cnt = length; cnt < currentArrayLength; cnt++) {
    arr.pop();
  }

  for (let cnt = currentArrayLength; cnt < length; cnt++) {
    arr.push(placeholder);
  }

  return arr;
}
