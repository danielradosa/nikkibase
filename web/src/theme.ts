import type { ThemeConfig } from 'antd'

const rose = '#d9538a'
const roseDeep = '#b93c6f'
const gold = '#d9a441'
const ink = '#4a2c3a'
const inkSoft = '#8b6b78'
const body = "'Nunito Sans Variable', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif"

export function themeFor(phone: boolean): ThemeConfig {
  return {
    token: {
      colorPrimary: rose,
      colorLink: rose,
      colorInfo: rose,
      colorSuccess: '#5fb99b',
      colorWarning: gold,
      colorError: '#d65b6a',
      colorText: ink,
      colorTextSecondary: inkSoft,
      colorTextDescription: inkSoft,
      colorBorder: 'rgba(217, 83, 138, 0.18)',
      colorBorderSecondary: 'rgba(217, 83, 138, 0.10)',
      colorBgContainer: '#ffffff',
      colorBgElevated: '#ffffff',
      colorBgLayout: 'transparent',
      fontFamily: body,
      borderRadius: 12,
      controlHeight: phone ? 44 : 38,
      wireframe: false,
    },
    components: {
      Layout: {
        headerBg: 'transparent',
        bodyBg: 'transparent',
        footerBg: 'transparent',
        headerHeight: 'auto' as unknown as number,
      },
      Table: {
        headerBg: '#fff8fa',
        rowHoverBg: '#fff8fa',
        borderColor: 'rgba(217, 83, 138, 0.10)',
        ...(phone ? { controlInteractiveSize: 19 } : {}),
      },
      Segmented: {
        itemSelectedBg: rose,
        itemSelectedColor: '#ffffff',
        trackBg: '#fdeef4',
        itemHoverBg: '#f9dce8',
        ...(phone ? { controlHeight: 48 } : {}),
      },
      Tabs: {
        inkBarColor: rose,
        itemSelectedColor: roseDeep,
        itemHoverColor: rose,
        titleFontSize: 15,
        ...(phone ? { horizontalItemPadding: '10px 0' } : {}),
      },
      Progress: { defaultColor: rose },
      Spin: { dotSize: 20, dotSizeSM: 14, dotSizeLG: 32 },
      Statistic: { colorTextDescription: inkSoft },
      Alert: { borderRadiusLG: 14 },
      Select: { optionSelectedBg: '#fdeef4' },
      Upload: { colorBorder: '#f2c2d6' },
    },
  }
}
