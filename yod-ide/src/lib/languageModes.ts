import type { Extension } from "@codemirror/state";
import { javascript } from "@codemirror/lang-javascript";
import { json } from "@codemirror/lang-json";
import { html } from "@codemirror/lang-html";
import { css } from "@codemirror/lang-css";
import { php } from "@codemirror/lang-php";
import { sql, MySQL, PostgreSQL, StandardSQL } from "@codemirror/lang-sql";
import { markdown } from "@codemirror/lang-markdown";
import { xml } from "@codemirror/lang-xml";
import { python } from "@codemirror/lang-python";
import { yaml } from "@codemirror/lang-yaml";
import { pathBase, isYodFamilyFile } from "./paths";

/** בוחר חבילת שפה לפי סיומת קובץ (לא יוד). */
export function languageSupportForPath(pathOrName: string | null | undefined): Extension | null {
  if (!pathOrName || isYodFamilyFile(pathOrName)) return null;
  const name = pathBase(pathOrName).toLowerCase();
  const ext = (() => {
    const i = name.lastIndexOf(".");
    return i >= 0 ? name.slice(i + 1) : "";
  })();

  switch (ext) {
    case "js":
    case "mjs":
    case "cjs":
      return javascript();
    case "jsx":
      return javascript({ jsx: true });
    case "ts":
      return javascript({ typescript: true });
    case "tsx":
      return javascript({ jsx: true, typescript: true });
    case "json":
    case "jsonc":
      return json();
    case "html":
    case "htm":
    case "xhtml":
      return html();
    case "css":
      return css();
    case "scss":
    case "less":
      return css();
    case "php":
    case "phtml":
      return php();
    case "sql":
      return sql({ dialect: StandardSQL });
    case "mysql":
      return sql({ dialect: MySQL });
    case "pgsql":
    case "psql":
      return sql({ dialect: PostgreSQL });
    case "md":
    case "markdown":
      return markdown();
    case "xml":
    case "svg":
      return xml();
    case "py":
      return python();
    case "yml":
    case "yaml":
      return yaml();
    default:
      return null;
  }
}
