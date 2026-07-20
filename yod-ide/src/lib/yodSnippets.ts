import { snippetCompletion, type Completion } from "@codemirror/autocomplete";

type SnippetDef = { keyword: string; template: string };

/**
 * תבניות קוד בעברית — הרחבה לתבנית מלאה עם קפיצה בין שדות (Tab).
 * שדות: `${שם}` שדה עם ערך ברירת מחדל, `${}` שדה ריק. הסדר לפי מיקום.
 */
const DEFS: SnippetDef[] = [
  { keyword: "פונקציה", template: "פונקציה ${שם}(${פרמטרים}) {\n\t${}\n}" },
  { keyword: "מחלקה", template: "מחלקה ${שם} {\n\tחדש(${}) {\n\t\t${}\n\t}\n}" },
  { keyword: "אם", template: "אם (${תנאי}) {\n\t${}\n}" },
  { keyword: "אחרת_אם", template: "אחרת_אם (${תנאי}) {\n\t${}\n}" },
  { keyword: "אחרת", template: "אחרת {\n\t${}\n}" },
  { keyword: "עבור", template: "עבור (${פריט} בתוך ${רשימה}) {\n\t${}\n}" },
  { keyword: "כל_עוד", template: "כל_עוד (${תנאי}) {\n\t${}\n}" },
  { keyword: "נסה", template: "נסה {\n\t${}\n} תפוס (${שגיאה}) {\n\t${}\n}" },
  { keyword: "בחר", template: "בחר (${ערך}) {\n\tמקרה ${}:\n\t\t${}\n\tברירת_מחדל:\n}" },
  { keyword: "משימה", template: "משימה {\n\t${}\n}" },
  { keyword: "הדפס", template: "הדפס(${})" },
];

/** שמות מילות המפתח שיש להן תבנית — לצורך מניעת כפילות בהשלמה. */
export const YOD_SNIPPET_KEYWORDS = new Set<string>(DEFS.map((d) => d.keyword));

/** רשימת השלמות התבניות (snippets) של יוד. */
export const yodSnippets: Completion[] = DEFS.map((d) =>
  snippetCompletion(d.template, {
    label: d.keyword,
    detail: "תבנית",
    type: "snippet",
    boost: 96,
  })
);
