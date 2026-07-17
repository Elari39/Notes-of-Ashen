export const MAX_ARTICLE_CONTENT_BYTES = 5 * 1024 * 1024;
export const MAX_TEXT_FIELD_BYTES = 65_535;

export const utf8ByteLength = (value: string): number => new TextEncoder().encode(value).byteLength;
