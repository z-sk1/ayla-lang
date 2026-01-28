import * as vscode from "vscode";
import { LanguageClient } from "vscode-languageclient/node";
import * as path from "path";

let client: LanguageClient;

export function activate(ctx: vscode.ExtensionContext) {
    const serverExe = ctx.asAbsolutePath("server/ayla-lsp");

    client = new LanguageClient(
        "ayla",
        "elen",
        {
            command: serverExe,
            transport: 0 // stdio
        },
        {
            documentSelector: [{ language: "ayla" }]
        }
    );

    client.start();
}

export function deactivate() {
    if (client) {
        return client.stop();
    }
}