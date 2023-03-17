export const themeData = JSON.parse("{\"home\":\"/guide/\",\"sidebarDepth\":2,\"navbar\":[{\"text\":\"主页\",\"link\":\"/\"},{\"text\":\"使用文档\",\"link\":\"/guide/\"},{\"text\":\"服务提供者\",\"link\":\"/provider/\"},{\"text\":\"Github\",\"link\":\"https://github.com/gohade/hade\"}],\"sidebar\":{\"/guide/\":[{\"title\":\"指南\",\"collapsable\":false,\"children\":[\"introduce\",\"install\",\"version\",\"build\",\"structure\",\"app\",\"env\",\"dev\",\"command\",\"cron\",\"middleware\",\"swagger\",\"provider\",\"model\",\"deploy\",\"util\",\"todo\"]}],\"/provider/\":[{\"title\":\"服务提供者\",\"collapsable\":false,\"children\":[\"app\",\"env\",\"config\",\"log\"]}]},\"locales\":{\"/\":{\"selectLanguageName\":\"English\"}},\"colorMode\":\"auto\",\"colorModeSwitch\":true,\"logo\":null,\"repo\":null,\"selectLanguageText\":\"Languages\",\"selectLanguageAriaLabel\":\"Select language\",\"editLink\":true,\"editLinkText\":\"Edit this page\",\"lastUpdated\":true,\"lastUpdatedText\":\"Last Updated\",\"contributors\":true,\"contributorsText\":\"Contributors\",\"notFound\":[\"There's nothing here.\",\"How did we get here?\",\"That's a Four-Oh-Four.\",\"Looks like we've got some broken links.\"],\"backToHome\":\"Take me home\",\"openInNewWindow\":\"open in new window\",\"toggleColorMode\":\"toggle color mode\",\"toggleSidebar\":\"toggle sidebar\"}")

if (import.meta.webpackHot) {
  import.meta.webpackHot.accept()
  if (__VUE_HMR_RUNTIME__.updateThemeData) {
    __VUE_HMR_RUNTIME__.updateThemeData(themeData)
  }
}

if (import.meta.hot) {
  import.meta.hot.accept(({ themeData }) => {
    __VUE_HMR_RUNTIME__.updateThemeData(themeData)
  })
}
