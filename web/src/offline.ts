export function keepForOffline(): void {
  if (!import.meta.env.PROD || !('serviceWorker' in navigator)) return
  const register = () => {
    navigator.serviceWorker.register('/sw.js').catch(() => {})
  }
  if (document.readyState === 'complete') register()
  else window.addEventListener('load', register, { once: true })
}
