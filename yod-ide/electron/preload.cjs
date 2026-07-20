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
  windowMinimize: () => ipcRenderer.invoke("win:minimize"),
  windowMaximizeToggle: () => ipcRenderer.invoke("win:maximizeToggle"),
  windowClose: () => ipcRenderer.invoke("win:close"),
  windowIsMaximized: () => ipcRenderer.invoke("win:isMaximized"),
  onMaximized: (cb) => {
    const handler = (_e, v) => cb(!!v);
    ipcRenderer.on("win:maximized", handler);
    return () => ipcRenderer.removeListener("win:maximized", handler);
  },
  runYod: (args, cwd) => ipcRenderer.invoke("yod:run", args, cwd),
  terminal: {
    create: (opts) => ipcRenderer.invoke("term:create", opts),
    write: (id, data) => ipcRenderer.invoke("term:input", { id, data }),
    resize: (id, cols, rows) => ipcRenderer.invoke("term:resize", { id, cols, rows }),
    kill: (id) => ipcRenderer.invoke("term:kill", { id }),
    onData: (cb) => {
      const handler = (_e, payload) => cb(payload);
      ipcRenderer.on("term:data", handler);
      return () => ipcRenderer.removeListener("term:data", handler);
    },
    onExit: (cb) => {
      const handler = (_e, payload) => cb(payload);
      ipcRenderer.on("term:exit", handler);
      return () => ipcRenderer.removeListener("term:exit", handler);
    },
  },
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
