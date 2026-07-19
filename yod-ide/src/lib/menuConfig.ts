export type MenuActionItem = {
  type?: "item";
  label: string;
  action: string;
  kb?: string;
};

export type MenuSeparator = { type: "separator" };

export type MenuEntry = MenuActionItem | MenuSeparator;

export type MenuGroup = {
  id: string;
  label: string;
  items: MenuEntry[];
};

/** אותו מבנה כמו העורך הישן. */
export const APP_MENU: MenuGroup[] = [
  {
    id: "file",
    label: "קובץ",
    items: [
      { label: "חדש", action: "file.new", kb: "Ctrl+N" },
      { label: "פתח…", action: "file.open", kb: "Ctrl+O" },
      { label: "פתח תיקייה…", action: "file.openFolder", kb: "Ctrl+Shift+O" },
      { label: "סגור תיקייה", action: "file.closeFolder" },
      { type: "separator" },
      { label: "שמור", action: "file.save", kb: "Ctrl+S" },
      { label: "שמור בשם…", action: "file.saveAs" },
      { label: "סגור טאב", action: "file.closeTab", kb: "Ctrl+W" },
      { type: "separator" },
      { label: "יציאה", action: "app.quit" },
    ],
  },
  {
    id: "tree",
    label: "סייר",
    items: [
      { label: "רענון עץ", action: "tree.refresh" },
      { type: "separator" },
      { label: "קובץ חדש", action: "tree.newFile" },
      { label: "תיקייה חדשה", action: "tree.newFolder" },
      { label: "שינוי שם", action: "tree.rename" },
      { label: "מחק", action: "tree.delete" },
      { type: "separator" },
      { label: "הצג בסייר Windows", action: "tree.reveal" },
    ],
  },
  {
    id: "edit",
    label: "עריכה",
    items: [
      { label: "בטל", action: "edit.undo", kb: "Ctrl+Z" },
      { label: "בצע שוב", action: "edit.redo", kb: "Ctrl+Y" },
      { type: "separator" },
      { label: "בחר הכל", action: "edit.selectAll", kb: "Ctrl+A" },
      { type: "separator" },
      { label: "חיפוש…", action: "edit.find", kb: "Ctrl+F" },
      { label: "החלפה…", action: "edit.replace", kb: "Ctrl+H" },
      { label: "מעבר לשורה…", action: "edit.goto", kb: "Ctrl+G" },
      { type: "separator" },
      { label: "מעבר להגדרה", action: "nav.gotoDef", kb: "F12 / Ctrl+לחיצה" },
      { label: "מצא הפניות", action: "nav.findRefs", kb: "Shift+F12" },
      { label: "זוג תואם (התחלה/סוף)", action: "edit.matchPair", kb: "Ctrl+}" },
      { type: "separator" },
      { label: "הערה / ביטול הערה", action: "edit.comment", kb: "Ctrl+/" },
      { label: "שכפול שורה", action: "edit.duplicate", kb: "Ctrl+Shift+D" },
    ],
  },
  {
    id: "code",
    label: "קוד",
    items: [
      { label: "קבע נקודה", action: "code.addBookmark", kb: "Ctrl+B" },
      { label: "נקודות — הצג לשונית", action: "code.showBookmarks" },
      { type: "separator" },
      { label: "נקה את כל הנקודות", action: "code.clearBookmarks" },
    ],
  },
  {
    id: "run",
    label: "הרצה",
    items: [
      { label: "הרץ (מפרש)", action: "run.interpreter", kb: "F5" },
      { label: "מכונה (bytecode)", action: "run.vm", kb: "F6" },
      { label: "בדוק קומפילציה", action: "run.check", kb: "F7" },
      { type: "separator" },
      { label: "ארוז ל־EXE", action: "run.pack", kb: "Ctrl+Shift+P" },
      { label: "פתח מיקום EXE", action: "run.openDistExe" },
    ],
  },
  {
    id: "view",
    label: "תצוגה",
    items: [
      { label: "סייר קבצים", action: "view.sidebar.files" },
      { label: "ניתוח קובץ", action: "view.sidebar.outline" },
      { label: "סימבולי פרויקט", action: "view.sidebar.symbols" },
      { type: "separator" },
      { label: "פתיחה מהירה…", action: "view.quickOpen", kb: "Ctrl+P" },
      { label: "חיפוש בפרויקט…", action: "view.findInFiles", kb: "Ctrl+Shift+F" },
      { type: "separator" },
      { label: "סדר קוד", action: "view.format", kb: "Shift+Alt+F" },
      { type: "separator" },
      { label: "הגדל גופן", action: "view.zoomIn", kb: "Ctrl+=" },
      { label: "הקטן גופן", action: "view.zoomOut", kb: "Ctrl+-" },
      { label: "איפוס גודל", action: "view.zoomReset", kb: "Ctrl+0" },
      { type: "separator" },
      { label: "לשונית בעיות", action: "view.problems" },
      { label: "לשונית פלט", action: "view.output" },
      { type: "separator" },
      { label: "פלטת פקודות", action: "view.palette", kb: "F1" },
    ],
  },
  {
    id: "help",
    label: "עזרה",
    items: [
      { label: "מדריך…", action: "help.guide" },
      { label: "קיצורי מקלדת…", action: "help.shortcuts" },
      { type: "separator" },
      { label: "אודות יוד…", action: "help.about" },
    ],
  },
];
