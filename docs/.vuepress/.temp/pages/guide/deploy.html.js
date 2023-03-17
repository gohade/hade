export const data = JSON.parse("{\"key\":\"v-8a14f834\",\"path\":\"/guide/deploy.html\",\"title\":\"自动部署\",\"lang\":\"en-US\",\"frontmatter\":{},\"headers\":[{\"level\":2,\"title\":\"SSH\",\"slug\":\"ssh\",\"link\":\"#ssh\",\"children\":[]},{\"level\":2,\"title\":\"deploy\",\"slug\":\"deploy\",\"link\":\"#deploy\",\"children\":[{\"level\":3,\"title\":\"部署前端\",\"slug\":\"部署前端\",\"link\":\"#部署前端\",\"children\":[]},{\"level\":3,\"title\":\"部署后端\",\"slug\":\"部署后端\",\"link\":\"#部署后端\",\"children\":[]},{\"level\":3,\"title\":\"前后端一起部署\",\"slug\":\"前后端一起部署\",\"link\":\"#前后端一起部署\",\"children\":[]},{\"level\":3,\"title\":\"部署回滚\",\"slug\":\"部署回滚\",\"link\":\"#部署回滚\",\"children\":[]}]}],\"git\":{\"updatedTime\":1671022339000,\"contributors\":[{\"name\":\"jianfengye\",\"email\":\"jianfengye110@gmail.com\",\"commits\":1}]},\"filePathRelative\":\"guide/deploy.md\"}")

if (import.meta.webpackHot) {
  import.meta.webpackHot.accept()
  if (__VUE_HMR_RUNTIME__.updatePageData) {
    __VUE_HMR_RUNTIME__.updatePageData(data)
  }
}

if (import.meta.hot) {
  import.meta.hot.accept(({ data }) => {
    __VUE_HMR_RUNTIME__.updatePageData(data)
  })
}
