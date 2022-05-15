/**
 * Generate a cryptographically secure random string using the specified number
 * of bytes then encode with base64.
 *
 * Base64 encodes 6 bits per character and pads with = so the length will not
 * equal the number of randomly generated bytes.
 * @see <https://developer.mozilla.org/en-US/docs/Glossary/Base64#encoded_size_increase>
 */
export const generateRandomString = (bytes: number): string => {
  const byteArr = window.crypto.getRandomValues(new Uint8Array(bytes))
  // The types for `map` don't seem to support mapping from one array type to
  // another and `String.fromCharCode.apply` wants `number[]` so loop like this
  // instead.
  const strArr: string[] = []
  for (const byte of byteArr) {
    strArr.push(String.fromCharCode(byte))
  }
  return btoa(strArr.join(""))
}
