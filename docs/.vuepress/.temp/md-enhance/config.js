import { defineClientConfig } from "@vuepress/client";
import CodeTabs from "/Users/jianfengye/Documents/workspace/gohade/hade/node_modules/vuepress-plugin-md-enhance/lib/client/components/CodeTabs.js";
import { hasGlobalComponent } from "/Users/jianfengye/Documents/workspace/gohade/hade/node_modules/vuepress-shared/lib/client/index.js";
import { CodeGroup, CodeGroupItem } from "/Users/jianfengye/Documents/workspace/gohade/hade/node_modules/vuepress-plugin-md-enhance/lib/client/compact/index.js";
import Mermaid from "/Users/jianfengye/Documents/workspace/gohade/hade/node_modules/vuepress-plugin-md-enhance/lib/client/components/Mermaid.js";

export default defineClientConfig({
  enhance: ({ app }) => {
    app.component("CodeTabs", CodeTabs);
    if(!hasGlobalComponent("CodeGroup", app)) app.component("CodeGroup", CodeGroup);
    if(!hasGlobalComponent("CodeGroupItem", app)) app.component("CodeGroupItem", CodeGroupItem);
    app.component("Mermaid", Mermaid);
  },
});
