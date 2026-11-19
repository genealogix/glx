import DefaultTheme from 'vitepress/theme'
import { inBrowser, withBase } from 'vitepress'
import './custom.css'

export default {
  extends: DefaultTheme,
  enhanceApp({ router }) {
    if (!inBrowser) return

    router.onBeforeRouteChange = (href) => {
      const target = new URL(href, location.href)
      const isConverter = (path) => path.replace(/\.html$/, '') === withBase('/convert')
      // An SPA navigation would leave already-running analytics on the page.
      if (isConverter(target.pathname) !== isConverter(location.pathname)) {
        location.assign(target.href)
        return false
      }
    }
  },
}
