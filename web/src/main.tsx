import './boot'
import { StrictMode, useMemo } from 'react'
import { createRoot } from 'react-dom/client'
import { ConfigProvider } from 'antd'
import 'antd/dist/reset.css'
import '@fontsource-variable/nunito-sans/opsz.css'
import '@fontsource/cormorant-garamond/500.css'
import '@fontsource/cormorant-garamond/600.css'
import '@fontsource/cormorant-garamond/700.css'
import '@fontsource/cormorant-garamond/600-italic.css'
import './styles.css'
import { themeFor } from './theme'
import { usePhone } from './usePhone'
import App from './App'
import Petals from './Petals'

const SPIN = { indicator: <Petals /> }

function Root() {
  const phone = usePhone()
  const theme = useMemo(() => themeFor(phone), [phone])
  return (
    <ConfigProvider theme={theme} spin={SPIN}>
      <App />
    </ConfigProvider>
  )
}

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <Root />
  </StrictMode>,
)
