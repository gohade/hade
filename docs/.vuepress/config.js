import {defaultTheme} from 'vuepress'
import {searchPlugin} from '@vuepress/plugin-search'
import {mdEnhancePlugin} from "vuepress-plugin-md-enhance";
import {copyCodePlugin} from "vuepress-plugin-copy-code2";
import {tocPlugin} from '@vuepress/plugin-toc'
import {googleAnalyticsPlugin} from '@vuepress/plugin-google-analytics'


export default {
    title: "hade框架", // 设置网站标题
    description: "一个支持前后端开发的基于协议的框架", //描述
    dest: "./dist/", // 设置输出目录
    port: 2333, //端口
    head: [["link", {rel: "icon", href: "/assets/img/head.png"}]],
    theme: defaultTheme({
        home: '/guide/',
        sidebarDepth: 2,
        //主题配置
        // logo: "/assets/img/head.png",
        // 添加导航栏
        navbar: [
            {text: "主页", link: "/"}, // 导航条
            {text: "使用文档", link: "/guide/"},
            {text: "服务提供者", link: "/provider/"},
            {text: "Github", link: "https://github.com/gohade/hade"}
        ],
        // 为以下路由添加侧边栏
        sidebar: {
            "/guide/": [
                {
                    title: "指南",
                    collapsable: false,
                    children: [
                        "introduce",
                        "install",
                        "version",
                        "build",
                        "structure",
                        "app",
                        "env",
                        "dev",
                        "command",
                        "cron",
                        "middleware",
                        "swagger",
                        "provider",
                        "model",
                        "deploy",
                        "util",
                        "todo",
                    ],
                },
            ],
            "/provider/": [
                {
                    title: "服务提供者",
                    collapsable: false,
                    children: [
                        "app",
                        "env",
                        "config",
                        "log",
                    ],
                },
            ],
        },
    }),
    plugins: [
        searchPlugin({}),
        mdEnhancePlugin({
            codetabs: true,
            mermaid: true,
        }),
        copyCodePlugin({
            locales: {
                '/': {
                    copy: "复制",
                    hint: "复制成功",
                }
            }
        }),
        tocPlugin({
            // 配置项
        }),
        googleAnalyticsPlugin({
            id: 'G-TTJKZDD7LR',
        }),

    ]
};

