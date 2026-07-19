/**
 * מדריך יוד — תפריט צד, חיפוש, וגלילה עוקבת
 */
(function () {
  "use strict";

  var VERSION = "0.75.0";

  /** עץ ניווט: פרקים + תת־סעיפים (עמודים) */
  var NAV = [
    {
      id: "start",
      title: "התחלה",
      keywords: "התקנה הרצה עורך מכונה ארוז פרויקט",
      children: [
        { title: "מדריך למתחילים", href: "עמודים/מתחילים.html", keywords: "אפס שלום פלט" },
        { title: "התחלה מהירה", href: "עמודים/התחלה.html", keywords: "build exe הרץ" },
        { title: "קובץ יחיד או פרויקט", href: "עמודים/קובץ-או-פרויקט.html", keywords: "התחל תיקייה כלול קובץ יחיד" },
        { title: "עורך", href: "עמודים/עורך.html", keywords: "IDE Electron CodeMirror השלמה סייר פרויקט התחל yod-ide" },
        { title: "מכונה", href: "עמודים/מכונה.html", keywords: "bytecode VM" },
        { title: "מכונה מול מפרש", href: "עמודים/מכונה-מול-מפרש.html", keywords: "fallback תמיכה" },
        { title: "ארוז", href: "עמודים/ארוז.html", keywords: "exe הפצה" },
        { title: "קונסול / חבילות", href: "עמודים/קונסול-חבילות.html", keywords: "REPL חבילה" },
        { title: "סגנון רשמי", href: "עמודים/סגנון-רשמי.html", keywords: "בדוק lint סוף מינוס" },
        { title: "אופרטורים", href: "עמודים/אופרטורים.html", keywords: "חשבון וגם או השמה מינוס" },
        { title: "הערות", href: "עמודים/הערות.html", keywords: "// /* הערה בלוק" },
        { title: "זיכרון / סגירות", href: "עמודים/זיכרון.html", keywords: "GC הפניה" },
      ],
    },
    {
      id: "keywords",
      title: "מילות מפתח",
      keywords: "בקרה פונקציה מחלקה שגיאות",
      children: [
        { title: "משתנה", href: "עמודים/משתנה.html", keywords: "השמה ערך" },
        { title: "אם / אחרת", href: "עמודים/אם.html", keywords: "תנאי" },
        { title: "בחר / מקרה", href: "עמודים/בחר.html", keywords: "switch" },
        { title: "כל_עוד", href: "עמודים/כל_עוד.html", keywords: "לולאה while" },
        { title: "עבור / מ עד / בתוך", href: "עמודים/עבור.html", keywords: "foreach range מספור" },
        { title: "עצור / המשך", href: "עמודים/עצור-המשך.html", keywords: "break continue" },
        { title: "פונקציה / החזר", href: "עמודים/פונקציה.html", keywords: "סגירה" },
        { title: "מחלקה / חדש / זה", href: "עמודים/מחלקה.html", keywords: "OOP בנאי" },
        { title: "מרחיב / הורה", href: "עמודים/מרחיב.html", keywords: "ירושה" },
        { title: "פרטי / ציבורי", href: "עמודים/פרטי-ציבורי.html", keywords: "גישה" },
        { title: "כלול", href: "עמודים/כלול.html", keywords: "מודול פרויקט התחל" },
        { title: "יבא / יצא / מודול", href: "עמודים/יבא-יצא.html", keywords: "import export מתוך" },
        { title: "נסה / תפוס / זרוק", href: "עמודים/נסה-תפוס.html", keywords: "שגיאה" },
        { title: "טיפוסים / תוצאה", href: "עמודים/טיפוסים-תוצאה.html", keywords: "בדוק_טיפוסים" },
        { title: "משימה / המתן", href: "עמודים/משימה.html", keywords: "async במקביל" },
      ],
    },
    {
      id: "types",
      title: "טיפוסים",
      keywords: "מחרוזת מספר רשימה מילון",
      children: [
        { title: "מחרוזת", href: "עמודים/מחרוזת.html", keywords: "טקסט תבנית backtick" },
        { title: "מספר", href: "עמודים/מספר.html", keywords: "עיגול חזקה" },
        { title: "רשימה", href: "עמודים/רשימה.html", keywords: "מפה סנן מיון" },
        { title: "מילון", href: "עמודים/מילון.html", keywords: "מפתח ערך שמות סדר" },
        { title: "סדרה", href: "עמודים/סדרה.html", keywords: "enum ערכים" },
        { title: "אמת / שקר / ריק", href: "עמודים/אמת-שקר-ריק.html", keywords: "לוגיקה ??" },
      ],
    },
    {
      id: "builtins",
      title: "פונקציות מובנות",
      keywords: "הדפס קלט טווח אקראי",
      children: [
        { title: "הדפס", href: "עמודים/הדפס.html", keywords: "קונסול" },
        { title: "קלט", href: "עמודים/קלט.html", keywords: "משתמש" },
        { title: "טווח / אקראי / אורך", href: "עמודים/ספריית-בסיס.html", keywords: "בסיס גלובלי" },
      ],
    },
    {
      id: "libs",
      title: "ספריות",
      keywords: "כלול מודול",
      children: [
        { title: "בסיס", href: "עמודים/ספריית-בסיס.html", keywords: "הדפס קלט" },
        { title: "קבצים", href: "עמודים/ספריית-קבצים.html", keywords: "קריאה כתיבה zip" },
        { title: "JSON", href: "עמודים/ספריית-JSON.html", keywords: "פרסר" },
        { title: "זמן", href: "עמודים/ספריית-זמן.html", keywords: "תאריך שעון" },
        { title: "טיימרים", href: "עמודים/ספריית-טיימרים.html", keywords: "כל כמה פעם אחת טיימר" },
        { title: "משחק", href: "עמודים/ספריית-משחק.html", keywords: "התנגשות פיזיקה פריים סנייק פלטפורמר" },
        { title: "מתמטיקה", href: "עמודים/ספריית-מתמטיקה.html", keywords: "פי סינוס atan2 זווית" },
        { title: "מספרים", href: "עמודים/ספריית-מספרים.html", keywords: "פסיקים כסף זיכרון אחוז מפריד" },
        { title: "מערכת", href: "עמודים/ספריית-מערכת.html", keywords: "סביבה תהליך מעבד זיכרון כונן" },
        { title: "רשת", href: "עמודים/ספריית-רשת.html", keywords: "HTTP WebSocket שקע מולטיפלייר" },
        { title: "שכבות ממשק", href: "עמודים/שכבות-ממשק.html", keywords: "Win32 רכיבים משטח HUD" },
        { title: "SQL", href: "עמודים/ספריית-SQL.html", keywords: "sqlite mysql" },
        { title: "חלונות", href: "עמודים/ספריית-חלונות.html", keywords: "GUI דפדפן WebView2 אתר משטח משטח_GPU טבלה גרף הרץ_js שלח בהודעה" },
        { title: "GPU", href: "עמודים/ספריית-GPU.html", keywords: "OpenGL סצנה קוביה glTF טקסטורה נכסים ZIP משטח_GPU" },
        { title: "GUI היברידי", href: "עמודים/GUI-היברידי.html", keywords: "walk HWND היבריד החלפה וידגטים" },
        { title: "תלת", href: "עמודים/ספריית-תלת.html", keywords: "WebGL תלת ממד glTF מצלמה קוביה דפדפן" },
        { title: "עיצוב", href: "עמודים/ספריית-עיצוב.html", keywords: "Bootstrap HTML דפדפן פקד כפתור UI RTL" },
        { title: "פרסום", href: "עמודים/ספריית-פרסום.html", keywords: "פרסומת באנר כרזה מודעה מרוחק ads banner תוכנה חופשית" },
        { title: "רכיבים", href: "עמודים/ספריית-רכיבים.html", keywords: "כפתור סליידר מתג משטח custom dark תפריט הקשר" },
        { title: "גרפים", href: "עמודים/ספריית-גרפים.html", keywords: "גרף עמודות קו עוגה סדרות מקרא" },
        { title: "עכבר", href: "עמודים/ספריית-עכבר.html", keywords: "מיקום כפתור שמאל ימין אמצע קריאה" },
        { title: "מקלדת", href: "עמודים/ספריית-מקלדת.html", keywords: "סקרור מקש חצים WASD משחק" },
        { title: "אוטומציה", href: "עמודים/ספריית-אוטומציה.html", keywords: "RPA עכבר הקלד צילום מסך" },
        { title: "הקלטה", href: "עמודים/ספריית-הקלטה.html", keywords: "ffmpeg מסך מיקרופון mp4 הקלטה אזור" },
        { title: "שמע", href: "עמודים/ספריית-שמע.html", keywords: "wav mp3 אפקט מוזיקה" },
        { title: "וידאו", href: "עמודים/ספריית-וידאו.html", keywords: "mp4 נגן webview2" },
        { title: "תמונות", href: "עמודים/ספריית-תמונות.html", keywords: "png jpg טען שמור" },
        { title: "דיבור", href: "עמודים/ספריית-דיבור.html", keywords: "tts הקראה sapi" },
        { title: "ציור", href: "עמודים/ספריית-ציור.html", keywords: "PNG צורות" },
        { title: "הצפנה", href: "עמודים/ספריית-הצפנה.html", keywords: "sha md5 aes gcm pbkdf2" },
        { title: "תרגום", href: "עמודים/ספריית-תרגום.html", keywords: "i18n gettext po mo rtl רבים מגדר" },
        { title: "לוח", href: "עמודים/ספריית-לוח.html", keywords: "clipboard" },
        { title: "גיליון", href: "עמודים/ספריית-גיליון.html", keywords: "excel xlsx דוח" },
        { title: "אימייל", href: "עמודים/ספריית-אימייל.html", keywords: "smtp שלח מצורף gmail גימייל" },
        { title: "חומרה", href: "עמודים/ספריית-חומרה.html", keywords: "serial COM ארדואינו ESP32" },
      ],
    },
    {
      id: "build-apps",
      title: "מדריך בניית תוכנה",
      keywords: "פרויקט אפליקציה גיבוי מוניטור סנכרון EXE בנייה מדריך",
      children: [
        {
          title: "גיבוי",
          href: "עמודים/בניית-גיבוי.html",
          keywords: "סנכרון FTP מניפסט עיצוב אחסון פורקים ארוז תוכנה מלאה",
        },
        {
          title: "מוניטור",
          href: "עמודים/בניית-מוניטור.html",
          keywords: "אתרים HTTP ניטור URL תור עיצוב תרגום בדיקה",
        },
      ],
    },
  ];

  function basePrefix() {
    var b = document.body.getAttribute("data-guide-base");
    if (b == null || b === "") return ".";
    return b.replace(/\/$/, "");
  }

  function resolve(href) {
    if (!href) return "#";
    if (href.charAt(0) === "#") {
      if (document.body.getAttribute("data-guide-home") === "true") return href;
      return basePrefix() + "/מדריך שפת יוד.html" + href;
    }
    var base = basePrefix();
    if (base === "." || base === "") return href;
    return base + "/" + href;
  }

  function homeHref() {
    var base = basePrefix();
    if (base === "." || base === "") return "מדריך שפת יוד.html";
    return base + "/מדריך שפת יוד.html";
  }

  function currentFile() {
    var path = decodeURIComponent(location.pathname.replace(/\\/g, "/"));
    var parts = path.split("/");
    var file = parts[parts.length - 1] || "";
    if (!file || file.indexOf(".html") === -1) return "";
    // בעמודי פרט: עמודים/X.html
    if (parts.length >= 2 && decodeURIComponent(parts[parts.length - 2]) === "עמודים") {
      return "עמודים/" + file;
    }
    return file;
  }

  function norm(s) {
    return String(s || "")
      .toLowerCase()
      .replace(/\s+/g, " ")
      .trim();
  }

  function matches(q, title, keywords) {
    if (!q) return true;
    var hay = norm(title + " " + (keywords || ""));
    return hay.indexOf(q) !== -1;
  }

  function pageSections(href) {
    var all = window.GUIDE_SECTIONS || {};
    return all[href] || [];
  }

  function matchingSections(q, href, liveHeads) {
    if (!q) return [];
    var secs = pageSections(href).slice();
    if (liveHeads && liveHeads.length) {
      var byTitle = {};
      for (var i = 0; i < secs.length; i++) byTitle[secs[i].title] = secs[i];
      for (var j = 0; j < liveHeads.length; j++) {
        var lh = liveHeads[j];
        if (byTitle[lh.title]) {
          if (lh.id) byTitle[lh.title].id = lh.id;
        } else {
          var row = { title: lh.title, id: lh.id || "", level: lh.level };
          secs.push(row);
          byTitle[lh.title] = row;
        }
      }
    }
    var out = [];
    for (var k = 0; k < secs.length; k++) {
      if (matches(q, secs[k].title, "")) out.push(secs[k]);
    }
    return out;
  }

  function renderSectionLinks(pageHref, sections, pageActive) {
    if (!sections || !sections.length) return "";
    var html = '<div class="nav-sub" data-page-toc="1">';
    for (var h = 0; h < sections.length; h++) {
      var item = sections[h];
      var link =
        item.id && pageActive
          ? "#" + item.id
          : resolve(pageHref) + (item.id ? "#" + item.id : "");
      html +=
        '<a href="' +
        link +
        '"' +
        (item.id && pageActive ? ' data-heading="' + item.id + '"' : "") +
        (item.level === "h3" ? ' style="padding-right:18px"' : "") +
        ' class="nav-section-link">' +
        item.title +
        "</a>";
    }
    html += "</div>";
    return html;
  }

  function ensureHeadingIds(article) {
    if (!article) return [];
    var heads = article.querySelectorAll("h2, h3");
    var items = [];
    for (var i = 0; i < heads.length; i++) {
      var h = heads[i];
      if (!h.id) {
        var raw = (h.textContent || "סעיף").trim();
        var id = "s-" + raw.replace(/\s+/g, "-").replace(/[^\w\u0590-\u05FF\-]/g, "");
        if (!id || id === "s-") id = "s-" + (i + 1);
        var base = id;
        var n = 2;
        while (document.getElementById(id)) {
          id = base + "-" + n++;
        }
        h.id = id;
      }
      items.push({ id: h.id, title: (h.textContent || "").trim(), level: h.tagName.toLowerCase() });
    }
    return items;
  }

  function buildSidebar() {
    var aside = document.getElementById("guide-sidebar");
    if (!aside) return null;

    var cur = currentFile();
    var isHome = document.body.getAttribute("data-guide-home") === "true";
    var articleHeads = ensureHeadingIds(document.querySelector(".article"));

    aside.innerHTML =
      '<div class="guide-sidebar-head">' +
      '<a class="guide-logo" href="' +
      homeHref() +
      '">' +
      '<span class="guide-logo-mark" aria-hidden="true">י</span>' +
      '<span class="guide-logo-text"><strong>מדריך יוד</strong><span>גרסה ' +
      VERSION +
      "</span></span>" +
      "</a>" +
      '<div class="guide-search">' +
      '<input type="search" id="guide-search-input" placeholder="חיפוש בעמודים ובסעיפים…" autocomplete="off" />' +
      '<span class="guide-search-icon" aria-hidden="true">⌕</span>' +
      "</div>" +
      "</div>" +
      '<nav class="guide-nav" id="guide-nav" aria-label="תוכן המדריך"></nav>' +
      '<div class="guide-sidebar-foot">גלילה עוקבת · חיפוש חי</div>';

    renderNav("", cur, isHome, articleHeads);
    return aside;
  }

  function renderNav(query, cur, isHome, articleHeads) {
    var nav = document.getElementById("guide-nav");
    if (!nav) return;
    var q = norm(query);
    var html = "";
    var any = false;

    for (var c = 0; c < NAV.length; c++) {
      var chapter = NAV[c];
      var childHtml = "";
      var chapterHit = matches(q, chapter.title, chapter.keywords);
      var open = false;
      var hasActivePage = false;

      for (var i = 0; i < chapter.children.length; i++) {
        var page = chapter.children[i];
        var pageActive = cur === page.href;
        var liveHeads = pageActive ? articleHeads : null;
        var sectionHits = q ? matchingSections(q, page.href, liveHeads) : [];
        var hit =
          chapterHit ||
          matches(q, page.title, page.keywords) ||
          sectionHits.length > 0;
        if (q && !hit) continue;
        if (!q || hit) {
          any = true;
        }
        var href = resolve(page.href);
        if (pageActive) {
          hasActivePage = true;
          open = true;
        }

        childHtml +=
          '<a class="nav-link' +
          (pageActive ? " active" : "") +
          '" href="' +
          href +
          '" data-page="' +
          page.href +
          '">' +
          page.title +
          "</a>";

        if (q && sectionHits.length) {
          childHtml += renderSectionLinks(page.href, sectionHits, pageActive);
        } else if (!q && pageActive && articleHeads && articleHeads.length) {
          childHtml += renderSectionLinks(page.href, articleHeads, true);
        }
      }

      if (isHome) {
        if (q && !chapterHit && !childHtml) {
          continue;
        }
        any = true;
        childHtml =
          '<a class="nav-link" href="#' +
          chapter.id +
          '" data-section="' +
          chapter.id +
          '">סקירת הפרק</a>' +
          childHtml;
        open = true;
      }

      if (q && chapterHit && !childHtml) {
        for (var j = 0; j < chapter.children.length; j++) {
          var p2 = chapter.children[j];
          childHtml +=
            '<a class="nav-link" href="' +
            resolve(p2.href) +
            '" data-page="' +
            p2.href +
            '">' +
            p2.title +
            "</a>";
        }
        any = true;
        open = true;
      }

      if (!childHtml) continue;
      if (!isHome && !q && hasActivePage) open = true;
      if (!isHome && !q && !hasActivePage) open = false;
      if (q) open = true;

      html +=
        '<div class="nav-chapter' +
        (open ? " open" : "") +
        '" data-chapter="' +
        chapter.id +
        '">' +
        '<button type="button" class="nav-chapter-btn" aria-expanded="' +
        (open ? "true" : "false") +
        '">' +
        "<span>" +
        chapter.title +
        '</span><span class="chev">▾</span>' +
        "</button>" +
        '<div class="nav-chapter-links">' +
        childHtml +
        "</div></div>";
    }

    if (!any) {
      html = '<div class="nav-empty">לא נמצאו תוצאות</div>';
    }
    nav.innerHTML = html;

    // פתיחה/סגירה
    var btns = nav.querySelectorAll(".nav-chapter-btn");
    for (var b = 0; b < btns.length; b++) {
      btns[b].addEventListener("click", function (ev) {
        var ch = ev.currentTarget.closest(".nav-chapter");
        if (!ch) return;
        ch.classList.toggle("open");
        ev.currentTarget.setAttribute(
          "aria-expanded",
          ch.classList.contains("open") ? "true" : "false"
        );
      });
    }
  }

  function setupSearch(articleHeads) {
    var input = document.getElementById("guide-search-input");
    if (!input) return;
    var cur = currentFile();
    var isHome = document.body.getAttribute("data-guide-home") === "true";
    var t = null;
    input.addEventListener("input", function () {
      clearTimeout(t);
      t = setTimeout(function () {
        renderNav(input.value, cur, isHome, articleHeads);
        setupScrollSpy();
      }, 80);
    });
  }

  function setupScrollSpy() {
    var headingLinks = document.querySelectorAll(".nav-sub a[data-heading]");
    var sectionLinks = document.querySelectorAll(".nav-link[data-section]");

    function clearActive(list) {
      for (var i = 0; i < list.length; i++) list[i].classList.remove("active");
    }

    function onScroll() {
      if (headingLinks.length) {
        var best = null;
        var bestTop = -Infinity;
        for (var i = 0; i < headingLinks.length; i++) {
          var id = headingLinks[i].getAttribute("data-heading");
          var el = document.getElementById(id);
          if (!el) continue;
          var top = el.getBoundingClientRect().top;
          if (top <= 96 && top > bestTop) {
            bestTop = top;
            best = headingLinks[i];
          }
        }
        clearActive(headingLinks);
        if (best) best.classList.add("active");
      }

      if (sectionLinks.length && document.body.getAttribute("data-guide-home") === "true") {
        var bestS = null;
        var bestST = -Infinity;
        for (var s = 0; s < sectionLinks.length; s++) {
          var sid = sectionLinks[s].getAttribute("data-section");
          var sec = document.getElementById(sid);
          if (!sec) continue;
          var st = sec.getBoundingClientRect().top;
          if (st <= 120 && st > bestST) {
            bestST = st;
            bestS = sectionLinks[s];
          }
        }
        clearActive(sectionLinks);
        if (bestS) bestS.classList.add("active");

        // פתח את הפרק הפעיל
        if (bestS) {
          var ch = bestS.closest(".nav-chapter");
          if (ch) ch.classList.add("open");
        }
      }
    }

    window.removeEventListener("scroll", window.__yodGuideScroll);
    window.__yodGuideScroll = onScroll;
    window.addEventListener("scroll", onScroll, { passive: true });
    onScroll();
  }

  function setupMobile() {
    var btn = document.querySelector(".sidebar-toggle");
    var backdrop = document.querySelector(".sidebar-backdrop");
    function close() {
      document.body.classList.remove("sidebar-open");
    }
    function toggle() {
      document.body.classList.toggle("sidebar-open");
    }
    if (btn) btn.addEventListener("click", toggle);
    if (backdrop) backdrop.addEventListener("click", close);
    document.addEventListener("keydown", function (e) {
      if (e.key === "Escape") close();
    });
    // סגירה אחרי לחיצה על קישור במובייל
    var aside = document.getElementById("guide-sidebar");
    if (aside) {
      aside.addEventListener("click", function (e) {
        var a = e.target.closest("a");
        if (a && window.matchMedia("(max-width: 980px)").matches) close();
      });
    }
  }

  function ensureShell() {
    document.body.classList.add("has-sidebar");
    if (!document.getElementById("guide-sidebar")) {
      var aside = document.createElement("aside");
      aside.id = "guide-sidebar";
      aside.className = "guide-sidebar";
      aside.setAttribute("aria-label", "תפריט מדריך");
      document.body.insertBefore(aside, document.body.firstChild);
    }
    // עטיפת תוכן קיים
    if (!document.querySelector(".guide-main")) {
      var main = document.createElement("div");
      main.className = "guide-main";
      var nodes = [];
      for (var i = 0; i < document.body.childNodes.length; i++) {
        var n = document.body.childNodes[i];
        if (n.nodeType === 1 && (n.id === "guide-sidebar" || n.classList.contains("sidebar-toggle") || n.classList.contains("sidebar-backdrop"))) {
          continue;
        }
        nodes.push(n);
      }
      for (var j = 0; j < nodes.length; j++) main.appendChild(nodes[j]);
      document.body.appendChild(main);
    }
    if (!document.querySelector(".sidebar-toggle")) {
      var t = document.createElement("button");
      t.type = "button";
      t.className = "sidebar-toggle";
      t.textContent = "תפריט המדריך";
      document.body.appendChild(t);
    }
    if (!document.querySelector(".sidebar-backdrop")) {
      var bd = document.createElement("div");
      bd.className = "sidebar-backdrop";
      document.body.appendChild(bd);
    }
  }

  function navScriptDir() {
    var list = document.querySelectorAll('script[src*="guide-nav.js"]');
    if (!list.length) return "js/";
    var src = list[list.length - 1].getAttribute("src") || "js/guide-nav.js";
    return src.replace(/guide-nav\.js(\?.*)?$/, "");
  }

  function loadSectionIndex(done) {
    if (window.GUIDE_SECTIONS) {
      done();
      return;
    }
    var scr = document.createElement("script");
    scr.src = navScriptDir() + "guide-sections.js";
    scr.onload = function () {
      done();
    };
    scr.onerror = function () {
      window.GUIDE_SECTIONS = window.GUIDE_SECTIONS || {};
      done();
    };
    document.head.appendChild(scr);
  }

  document.addEventListener("DOMContentLoaded", function () {
    loadSectionIndex(function () {
      ensureShell();
      var articleHeads = ensureHeadingIds(document.querySelector(".article"));
      buildSidebar();
      setupSearch(articleHeads);
      setupScrollSpy();
      setupMobile();
    });
  });
})();
