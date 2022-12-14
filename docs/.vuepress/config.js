module.exports = {
    title: "hade框架", // 设置网站标题
    description: "一个支持前后端开发的基于协议的框架", //描述
    dest: "./dist/", // 设置输出目录
    port: 2333, //端口
    base: "/",
    head: [["link", {rel: "icon", href: "/assets/img/head.png"}]],
    themeConfig: {
        //主题配置
        // logo: "/assets/img/head.png",
        // 添加导航栏
        nav: [
            {text: "主页", link: "/"}, // 导航条
            {text: "使用文档", link: "/guide/"},
            {text: "服务提供者", link: "/provider/"},
            {
                text: "github",
                // 这里是下拉列表展现形式。
                items: [
                    {
                        text: "hade",
                        link: "https://github.com/gohade/hade",
                    },
                ],
            },
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
    },
};

