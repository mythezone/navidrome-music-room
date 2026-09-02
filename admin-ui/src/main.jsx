import React from 'react'
import ReactDOM from 'react-dom'
import { CssBaseline, ThemeProvider, createMuiTheme } from '@material-ui/core'
import App from './App'

const theme = createMuiTheme({
  palette: {
    type: 'dark',
    primary: { main: '#90caf9' },
    secondary: { main: '#b39ddb' },
    background: { default: '#303030', paper: '#424242' },
    success: { main: '#81c784' },
    warning: { main: '#ffb74d' },
  },
  typography: {
    fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", "Noto Sans SC", sans-serif',
    h5: { fontWeight: 600 },
    h6: { fontWeight: 600 },
    button: { textTransform: 'none', fontWeight: 600 },
  },
  shape: { borderRadius: 8 },
  overrides: {
    MuiPaper: { rounded: { borderRadius: 10 } },
    MuiButton: { root: { borderRadius: 6 } },
    MuiChip: { root: { fontWeight: 600 } },
  },
})

ReactDOM.render(
  <ThemeProvider theme={theme}>
    <CssBaseline />
    <App />
  </ThemeProvider>,
  document.getElementById('root'),
)
