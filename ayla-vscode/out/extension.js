"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.activate = activate;
exports.deactivate = deactivate;
const node_1 = require("vscode-languageclient/node");
let client;
function activate(ctx) {
    const serverExe = ctx.asAbsolutePath("server/ayla-lsp");
    client = new node_1.LanguageClient("ayla", "elen", {
        command: serverExe,
        transport: 0 // stdio
    }, {
        documentSelector: [{ language: "ayla" }]
    });
    client.start();
}
function deactivate() {
    if (client) {
        return client.stop();
    }
}
//# sourceMappingURL=extension.js.map