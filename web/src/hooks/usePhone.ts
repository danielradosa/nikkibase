import { Grid } from 'antd'

export function usePhone(): boolean {
  return Grid.useBreakpoint().sm === false
}
