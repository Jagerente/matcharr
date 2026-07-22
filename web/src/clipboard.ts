// navigator.clipboard is secure-context only (LAN clients run plain HTTP)
// and its writeText can hang when the document is unfocused, hence the
// timeout race and the execCommand fallback.
export async function copyText(text: string): Promise<boolean> {
  if (navigator.clipboard && window.isSecureContext) {
    try {
      await Promise.race([
        navigator.clipboard.writeText(text),
        new Promise((_, reject) =>
          setTimeout(() => reject(new Error('clipboard timeout')), 1000),
        ),
      ])
      return true
    } catch {
      /* fall through to the legacy path */
    }
  }

  const textarea = document.createElement('textarea')
  textarea.value = text
  textarea.setAttribute('readonly', '')
  textarea.style.position = 'fixed'
  textarea.style.opacity = '0'
  document.body.appendChild(textarea)
  textarea.select()
  try {
    return document.execCommand('copy')
  } catch {
    return false
  } finally {
    textarea.remove()
  }
}
