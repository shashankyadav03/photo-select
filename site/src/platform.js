// Best-effort guess at the visitor's computer, so the right instructions open first.

export const PLATFORMS = {
  mac: 'Mac',
  windows: 'Windows',
  linux: 'Linux',
}

export function detectOS() {
  const ua = navigator.userAgent || ''
  const platform = navigator.userAgentData?.platform || navigator.platform || ''
  // iPads report themselves as Macs but have touch screens.
  if (/iPhone|iPad|iPod|Android/i.test(ua) || (/Mac/i.test(platform) && navigator.maxTouchPoints > 1)) {
    return 'phone'
  }
  if (/Win/i.test(platform) || /Windows/i.test(ua)) return 'windows'
  if (/Mac/i.test(platform) || /Mac OS X/i.test(ua)) return 'mac'
  if (/Linux|X11|CrOS/i.test(platform + ua)) return 'linux'
  return 'mac'
}

// Resolves to 'apple-silicon', 'intel' or null when the browser won't say.
export async function detectMacChip() {
  try {
    const hints = await navigator.userAgentData?.getHighEntropyValues?.(['architecture'])
    if (hints?.architecture === 'arm') return 'apple-silicon'
    if (hints?.architecture === 'x86') return 'intel'
  } catch {
    // fall through to the WebGL check
  }
  try {
    const gl = document.createElement('canvas').getContext('webgl')
    const ext = gl?.getExtension('WEBGL_debug_renderer_info')
    const renderer = ext ? gl.getParameter(ext.UNMASKED_RENDERER_WEBGL) : ''
    if (/Apple M\d/i.test(renderer)) return 'apple-silicon'
    if (/Intel|AMD|Radeon|NVIDIA/i.test(renderer)) return 'intel'
  } catch {
    // unknown
  }
  return null
}
