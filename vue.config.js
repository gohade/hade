// 配置参考https://cli.vuejs.org/zh/config/

const path = require('path');

function resolve(dir) {
  return path.join(__dirname, dir);
}

module.exports = {
  publicPath: "./",
  outputDir: "dist",
  assetsDir: "asset",
  indexPath: "index.html",
  filenameHashing: true,
  lintOnSave: true,
  runtimeCompiler: true,
  pages: {
    index: {
      entry: 'frontend/main.js',
      template: 'frontend/public/index.html',
      filename: 'index.html',
      chunks: ['chunk-vendors', 'chunk-common', 'index']
    }
  },
  chainWebpack: (config) => {
    config.resolve.alias
      .set('@', resolve('frontend'))
      .set('@assets', resolve('frontend/assets'))
  },
  configureWebpack: () => {

  },
  devServer: {
    host: '0.0.0.0',
    port: 8080,
    proxy: null,
  },
  "transpileDependencies": [
    "vuetify"
  ]
}