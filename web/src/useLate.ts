import { useEffect, useState } from 'react'

const APPEAR = 150

export function useLate(on: boolean): boolean {
  const [late, setLate] = useState(false)
  useEffect(() => {
    if (!on) {
      setLate(false)
      return
    }
    const t = setTimeout(() => setLate(true), APPEAR)
    return () => clearTimeout(t)
  }, [on])
  return on && late
}
