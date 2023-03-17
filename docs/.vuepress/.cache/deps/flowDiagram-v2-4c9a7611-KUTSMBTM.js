import {
  flowRendererV2,
  flowStyles
} from "./chunk-VAMLTNBJ.js";
import "./chunk-4JUMWCIN.js";
import {
  flowDb,
  parser$1
} from "./chunk-VVHDEELY.js";
import "./chunk-ISEDXLA7.js";
import "./chunk-BKFECFQP.js";
import "./chunk-QUPH2EAF.js";
import "./chunk-L7F6RK2W.js";
import "./chunk-F5B4Z2ER.js";
import "./chunk-6VCXTPHY.js";
import {
  require_dist
} from "./chunk-46H3B7RD.js";
import "./chunk-FFOQTTY5.js";
import "./chunk-42NOBOWJ.js";
import {
  require_dayjs_min,
  setConfig
} from "./chunk-5CVQM243.js";
import {
  __toESM
} from "./chunk-HYZYPRER.js";

// node_modules/mermaid/dist/flowDiagram-v2-4c9a7611.js
var import_sanitize_url = __toESM(require_dist(), 1);
var import_dayjs = __toESM(require_dayjs_min(), 1);
var diagram = {
  parser: parser$1,
  db: flowDb,
  renderer: flowRendererV2,
  styles: flowStyles,
  init: (cnf) => {
    if (!cnf.flowchart) {
      cnf.flowchart = {};
    }
    cnf.flowchart.arrowMarkerAbsolute = cnf.arrowMarkerAbsolute;
    setConfig({ flowchart: { arrowMarkerAbsolute: cnf.arrowMarkerAbsolute } });
    flowRendererV2.setConf(cnf.flowchart);
    flowDb.clear();
    flowDb.setGen("gen-2");
  }
};
export {
  diagram
};
//# sourceMappingURL=flowDiagram-v2-4c9a7611-KUTSMBTM.js.map
