const { contextBridge, ipcRenderer } = require("electron");

contextBridge.exposeInMainWorld("yod", {
  readDir: (dirPath) => ipcRenderer.invoke("fs:readDir", dirPath),
  readFile: (filePath) => ipcRenderer.invoke("fs:readFile", filePath),
  writeFile: (filePath, text) => ipcRenderer.invoke("fs:writeFile", filePath, text),
  mkdir: (dirPath) => ipcRenderer.invoke("fs:mkdir", dirPath),
  rename: (from, to) => ipcRenderer.invoke("fs:rename", from, to),
  remove: (p) => ipcRenderer.invoke("fs:remove", p),
  stat: (p) => ipcRenderer.invoke("fs:stat", p),
  exists: (p) => ipcRenderer.invoke("fs:exists", p),
  openFolder: () => ipcRenderer.invoke("dialog:openFolder"),
  openFileDialog: () => ipcRenderer.invoke("dialog:openFile"),
  saveFileDialog: (defaultPath) => ipcRenderer.invoke("dialog:saveFile", defaultPath),
  dialogPrompt: (opts) => ipcRenderer.invoke("dialog:prompt", opts),
  showItem: (p) => ipcRenderer.invoke("shell:showItem", p),
  openPath: (p) => ipcRenderer.invoke("shell:openPath", p),
  openExternal: (url) => ipcRenderer.invoke("shell:openExternal", url),
  writeClipboard: (text) => ipcRenderer.invoke("clipboard:writeText", text),
  openGuide: () => ipcRenderer.invoke("guide:open"),
  getPaths: () => ipcRenderer.invoke("app:getPaths"),
  quit: () => ipcRenderer.invoke("app:quit"),
  runYod: (args, cwd) => ipcRenderer.invoke("yod:run", args, cwd),
  onOpenPath: (cb) => {
    const handler = (_e, p) => cb(p);
    ipcRenderer.on("yod:open-path", handler);
    return () => ipcRenderer.removeListener("yod:open-path", handler);
  },
  onMenu: (cb) => {
    const handler = (_e, action) => cb(action);
    ipcRenderer.on("yod:menu", handler);
    return () => ipcRenderer.removeListener("yod:menu", handler);
  },
});
