/**
 * מילות מפתח של יוד — מסווגות לצבעי PHP Dark+.
 */
export const YOD_CONTROL = [
  "אם",
  "אחרת",
  "אחרת_אם",
  "כל_עוד",
  "כל עוד",
  "עבור",
  "החזר",
  "עצור",
  "שבור",
  "המשך",
  "נסה",
  "תפוס",
  "זרוק",
  "בתוך",
  "סוף",
  "וגם",
  "או",
  "ו",
  "בחר",
  "מקרה",
  "ברירת_מחדל",
] as const;

export const YOD_DECL_KEYWORDS = [
  "פונקציה",
  "מחלקה",
  "משתנה",
  "קבוע",
  "חדש",
  "מרחיב",
  "פרטי",
  "ציבורי",
  "כלול",
  "מודול",
  "יצא",
  "יבא",
  "מתוך",
  "סדרה",
  "לא",
  "הורה",
  "משימה",
  "המתן",
  "במקביל",
  "לבסוף",
  "מ",
  "עד",
  "בצע",
] as const;

export const YOD_CONSTANTS = ["אמת", "שקר", "ריק", "זה"] as const;

export const YOD_BUILTINS = ["הדפס", "קלט", "אורך", "טווח", "אקראי"] as const;

/** כל מילות המפתח (להשלמה / חיפוש). */
export const YOD_KEYWORDS = [
  ...YOD_CONTROL,
  ...YOD_DECL_KEYWORDS,
  ...YOD_CONSTANTS,
] as const;

export const yodControlSet = new Set<string>(YOD_CONTROL);
export const yodDeclKeywordSet = new Set<string>(YOD_DECL_KEYWORDS);
export const yodConstantSet = new Set<string>(YOD_CONSTANTS);
export const yodBuiltinSet = new Set<string>(YOD_BUILTINS);

export const yodAfterFunction = new Set(["פונקציה"]);
export const yodAfterClass = new Set(["מחלקה", "מרחיב", "חדש"]);
export const yodAfterVar = new Set(["משתנה", "קבוע"]);
