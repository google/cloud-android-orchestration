function setValue(
  root: object,
  selectors: string[],
  value: any,
  options: {ifexists: 'override' | 'skip'} = {ifexists: 'override'}
) {
  let traversal: any = root;
  for (const [i, selector] of selectors.entries()) {
    if (i !== selectors.length - 1) {
      if (!traversal[selector]) {
        traversal[selector] = {};
      }

      traversal = traversal[selector];
      continue;
    }

    if (!traversal[selector] || options.ifexists === 'override') {
      traversal[selector] = value;
    }
  }
}

function parse(s: string | undefined | null) {
  if (!s) {
    return {};
  }

  try {
    return JSON.parse(s);
  } catch {
    return {};
  }
}

function stringify(o: object) {
  return JSON.stringify(o, undefined, 2);
}

export default {setValue, parse, stringify};
