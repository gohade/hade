export const data = JSON.parse("{\"key\":\"v-4a6824e7\",\"path\":\"/guide/todo.html\",\"title\":\"待做事项\",\"lang\":\"en-US\",\"frontmatter\":{},\"headers\":[],\"git\":{\"updatedTime\":1644804900000,\"contributors\":[{\"name\":\"yejianfeng\",\"email\":\"jianfengye110@gmail.com\",\"commits\":2}]},\"filePathRelative\":\"guide/todo.md\"}")

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
