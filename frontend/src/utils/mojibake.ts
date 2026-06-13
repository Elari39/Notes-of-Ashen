const mojibakePattern = /[\u0080-\u009f\u00c0-\u00ff\u0152\u0153\u0160\u0161\u0178\u017d\u017e\u0192\u02c6\u02dc\u2018-\u201a\u201c-\u201e\u2020-\u2022\u2026\u2030\u2039\u203a\u20ac\u2122]/;
const cjkPattern = /[\u3400-\u9fff]/;
const utf8Decoder = typeof TextDecoder === 'undefined' ? null : new TextDecoder('utf-8', { fatal: true });

const windows1252SpecialBytes = new Map<string, number>([
  ['€', 0x80],
  ['‚', 0x82],
  ['ƒ', 0x83],
  ['„', 0x84],
  ['…', 0x85],
  ['†', 0x86],
  ['‡', 0x87],
  ['ˆ', 0x88],
  ['‰', 0x89],
  ['Š', 0x8a],
  ['‹', 0x8b],
  ['Œ', 0x8c],
  ['Ž', 0x8e],
  ['‘', 0x91],
  ['’', 0x92],
  ['“', 0x93],
  ['”', 0x94],
  ['•', 0x95],
  ['–', 0x96],
  ['—', 0x97],
  ['˜', 0x98],
  ['™', 0x99],
  ['š', 0x9a],
  ['›', 0x9b],
  ['œ', 0x9c],
  ['ž', 0x9e],
  ['Ÿ', 0x9f],
]);

export const fixVisibleMojibake = (value: string) => {
  if (!mojibakePattern.test(value)) {
    return value;
  }

  const decoded = decodeRepeatedly(value);
  if (decoded && looksBetter(decoded, value)) {
    return decoded;
  }

  const partiallyDecoded = decodeByteRuns(value);
  return partiallyDecoded !== value && looksBetter(partiallyDecoded, value) ? partiallyDecoded : value;
};

export const fixVisibleMojibakeDeep = <T>(value: T): T => fixDeep(value) as T;

const fixDeep = (value: unknown): unknown => {
  if (typeof value === 'string') {
    return fixVisibleMojibake(value);
  }
  if (Array.isArray(value)) {
    return value.map((item) => fixDeep(item));
  }
  if (!isPlainObject(value)) {
    return value;
  }

  return Object.entries(value).reduce<Record<string, unknown>>((next, [key, item]) => {
    next[key] = fixDeep(item);
    return next;
  }, {});
};

const decodeRepeatedly = (value: string) => {
  let current = value;
  let best = '';

  for (let index = 0; index < 3; index += 1) {
    const decoded = decodeWindows1252AsUtf8(current);
    if (!decoded || decoded === current) {
      break;
    }
    if (!best || looksBetter(decoded, best)) {
      best = decoded;
    }
    current = decoded;
  }

  return best || null;
};

const decodeByteRuns = (value: string) => {
  let result = '';
  let run = '';

  const flushRun = () => {
    if (!run) {
      return;
    }
    const decoded = mojibakePattern.test(run) ? decodeRepeatedly(run) : null;
    result += decoded && looksBetter(decoded, run) ? decoded : run;
    run = '';
  };

  for (const char of value) {
    if (byteFromWindows1252Char(char) === null) {
      flushRun();
      result += char;
    } else {
      run += char;
    }
  }
  flushRun();

  return result;
};

const decodeWindows1252AsUtf8 = (value: string) => {
  const bytes: number[] = [];
  for (const char of value) {
    const byte = byteFromWindows1252Char(char);
    if (byte === null) {
      return null;
    }
    bytes.push(byte);
  }
  return decodeUtf8Bytes(bytes);
};

const byteFromWindows1252Char = (char: string) => {
  const specialByte = windows1252SpecialBytes.get(char);
  if (specialByte !== undefined) {
    return specialByte;
  }

  const code = char.charCodeAt(0);
  return code <= 0xff ? code : null;
};

const decodeUtf8Bytes = (bytes: number[]) => {
  try {
    if (utf8Decoder) {
      return utf8Decoder.decode(new Uint8Array(bytes));
    }
    return decodeURIComponent(bytes.map((byte) => `%${byte.toString(16).padStart(2, '0')}`).join(''));
  } catch {
    return null;
  }
};

const looksBetter = (decoded: string, original: string) => {
  const originalScore = mojibakeScore(original);
  const decodedScore = mojibakeScore(decoded);
  return decodedScore < originalScore && cjkPattern.test(decoded);
};

const mojibakeScore = (value: string) => (value.match(mojibakePattern) || []).length;

const isPlainObject = (value: unknown): value is Record<string, unknown> => {
  return Object.prototype.toString.call(value) === '[object Object]';
};
