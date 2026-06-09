const mojibakePattern = /[ÃÂÄÅÆÇÈÉÐÑÒÓÔÕÖ×ØÙÚÛÜÝÞßàáâãäåæçèéêëìíîïðñòóôõöøùúûüýþÿ]/;

export const fixVisibleMojibake = (value: string) => {
  if (!mojibakePattern.test(value)) {
    return value;
  }

  try {
    const decoded = decodeURIComponent(escape(value));
    return looksBetter(decoded, value) ? decoded : value;
  } catch {
    return value;
  }
};

const looksBetter = (decoded: string, original: string) => {
  const originalScore = mojibakeScore(original);
  const decodedScore = mojibakeScore(decoded);
  return decodedScore < originalScore && /[\u4e00-\u9fff]/.test(decoded);
};

const mojibakeScore = (value: string) => (value.match(mojibakePattern) || []).length;
