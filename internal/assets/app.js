// RSSy page behaviors. Every page works without this file; it adds in-place
// actions (no reloads), undo, filters and keyboard shortcuts on top.
(function () {
  "use strict";

  var $ = function (s, r) { return (r || document).querySelector(s); };
  var $$ = function (s, r) { return Array.prototype.slice.call((r || document).querySelectorAll(s)); };

  function post(url, data) {
    var body = new URLSearchParams();
    Object.keys(data || {}).forEach(function (k) {
      var v = data[k];
      if (Array.isArray(v)) v.forEach(function (x) { body.append(k, x); });
      else body.append(k, v);
    });
    return fetch(url, {
      method: "POST",
      headers: { "X-Requested-With": "fetch", "Content-Type": "application/x-www-form-urlencoded" },
      body: body,
      credentials: "same-origin",
    }).then(function (r) {
      if (!r.ok) throw new Error("HTTP " + r.status);
      var ct = r.headers.get("Content-Type") || "";
      return ct.indexOf("json") >= 0 ? r.json() : {};
    });
  }

  // ---- toast + undo ----
  var undoFn = null, toastTimer = null;
  function toast(msg, undo) {
    var t = $("#toast");
    if (!t) {
      t = document.createElement("div");
      t.id = "toast"; t.className = "toast"; t.setAttribute("role", "status");
      document.body.appendChild(t);
    }
    t.textContent = "";
    var span = document.createElement("span");
    span.textContent = msg;
    t.appendChild(span);
    undoFn = undo || null;
    if (undo) {
      var b = document.createElement("button");
      b.type = "button"; b.textContent = "Undo (u)";
      b.onclick = runUndo;
      t.appendChild(b);
    }
    t.hidden = false;
    clearTimeout(toastTimer);
    toastTimer = setTimeout(function () { t.hidden = true; undoFn = null; }, 5000);
  }
  function runUndo() {
    if (!undoFn) return;
    var fn = undoFn; undoFn = null;
    var t = $("#toast"); if (t) t.hidden = true;
    fn();
  }
  function fail() { toast("Something went wrong. Reload and try again."); }
  window.rssy = { toast: toast, post: post };

  // ---- nav ----
  var path = location.pathname;
  $$(".navbar a").forEach(function (link) {
    var href = link.getAttribute("href");
    var active = href === "/"
      ? path === "/" || path.indexOf("/category/") === 0 || (path.indexOf("/article/") === 0 && !/[?&]from=favorites/.test(location.search))
      : path.indexOf(href) === 0 || (href === "/favorites" && path.indexOf("/article/") === 0 && /[?&]from=favorites/.test(location.search));
    if (href === "/feed" && path.indexOf("/feed/") === 0) active = true;
    link.classList.toggle("active", active);
    if (active) link.setAttribute("aria-current", "page");
  });

  // ---- help ----
  var help = $("#help");
  function showHelp() { if (help) help.hidden = false; }
  $$("[data-act=help]").forEach(function (b) { b.addEventListener("click", showHelp); });
  if (help) help.addEventListener("click", function (e) { if (e.target === help) help.hidden = true; });

  // ---- server messages ----
  $$("[data-toast]").forEach(function (el) {
    var msg = el.getAttribute("data-toast");
    if (msg) toast(msg);
    if (el.hasAttribute("data-strip")) {
      var url = new URL(location.href);
      el.getAttribute("data-strip").split(",").forEach(function (k) { url.searchParams.delete(k); });
      history.replaceState({}, "", url);
    }
  });

  // ---- visited links ----
  function visitedKey(a) { return a.href; }
  $$(".article-link").forEach(function (a) {
    try { if (localStorage.getItem(visitedKey(a))) a.classList.add("visited"); } catch (e) {}
    a.addEventListener("click", function () {
      try { localStorage.setItem(visitedKey(a), "1"); } catch (e) {}
      a.classList.add("visited");
    });
  });

  var keyHandlers = [];
  var gPending = false;
  document.addEventListener("keydown", function (e) {
    if (e.metaKey || e.ctrlKey || e.altKey) return;
    if (e.target.matches("input, select, textarea")) { if (e.key === "Escape") e.target.blur(); return; }
    if (help && !help.hidden) { help.hidden = true; e.preventDefault(); return; }
    var k = e.key;
    if (k === "?") { showHelp(); return; }
    if (gPending) {
      gPending = false;
      var dest = { h: "/", f: "/feed", s: "/ai-summary", e: "/explore", p: "/preference" }[k];
      if (dest) location.href = dest;
      return;
    }
    if (k === "g") { gPending = true; setTimeout(function () { gPending = false; }, 900); return; }
    if (k === "u" && undoFn) { runUndo(); return; }
    for (var i = 0; i < keyHandlers.length; i++) if (keyHandlers[i](e) === true) { e.preventDefault(); return; }
  });

  // =====================================================================
  // Article list (home / favorites / category / feed)
  // =====================================================================
  var list = $(".article-list");
  if (list) initList(list);

  function initList(list) {
    var page = list.getAttribute("data-page") || "home";
    var filter = "all";
    var cur = -1;
    var showHidden = $("#show_hidden") ? $("#show_hidden").checked : true;

    function rows() { return $$(".article-item", list).filter(function (r) { return !r.classList.contains("removed"); }); }
    function visibleRows() { return rows().filter(function (r) { return !r.hidden; }); }
    function matches(r) {
      if (!showHidden && r.getAttribute("data-hidden") === "1") return false;
      if (filter === "fav") return r.getAttribute("data-fav") === "1";
      if (filter !== "all") return r.getAttribute("data-cat") === filter;
      return true;
    }
    function base() { return rows().filter(function (r) { return showHidden || r.getAttribute("data-hidden") !== "1"; }); }

    function refresh() {
      rows().forEach(function (r) { r.hidden = !matches(r); });
      $$(".timeline-divider", list).forEach(function (d) {
        var g = d.getAttribute("data-group");
        var n = visibleRows().filter(function (r) { return r.getAttribute("data-group") === g; }).length;
        d.hidden = n === 0;
        var c = $(".n", d); if (c) c.textContent = n;
      });
      var b = base();
      var count = $("#list-count"); if (count) count.textContent = page === "favorites" ? rows().length : b.length;
      $$("[data-filter]").forEach(function (chip) {
        var f = chip.getAttribute("data-filter");
        var n = f === "all" ? b.length
          : f === "fav" ? b.filter(function (r) { return r.getAttribute("data-fav") === "1"; }).length
          : b.filter(function (r) { return r.getAttribute("data-cat") === f; }).length;
        var nEl = $(".n", chip); if (nEl) nEl.textContent = n;
        chip.hidden = f !== "all" && n === 0 && filter !== f;
        chip.classList.toggle("active", f === filter);
        chip.setAttribute("aria-pressed", f === filter ? "true" : "false");
      });
      var hn = $("#hidden-count");
      if (hn) {
        var hc = rows().filter(function (r) { return r.getAttribute("data-hidden") === "1"; }).length;
        hn.textContent = hc;
        hn.closest("form").hidden = hc === 0 && !showHidden;
      }
      var vis = visibleRows();
      var all = $("[data-act=allread]");
      if (all) {
        all.disabled = vis.filter(function (r) { return r.getAttribute("data-read") !== "1"; }).length === 0;
        $(".lbl", all).textContent = filter === "all" && vis.length === base().length ? "all" : vis.length;
      }
      var empty = $("#list-empty"); if (empty) empty.hidden = vis.length !== 0;
      if (cur >= vis.length) cur = vis.length - 1;
      paintCur(false);
    }

    function paintCur(scroll) {
      var vis = visibleRows();
      rows().forEach(function (r) { r.classList.remove("cur"); });
      if (cur >= 0 && vis[cur]) {
        vis[cur].classList.add("cur");
        if (scroll) vis[cur].scrollIntoView({ block: "nearest" });
      }
    }

    // filter chips
    $$("[data-filter]").forEach(function (chip) {
      chip.addEventListener("click", function () {
        filter = chip.getAttribute("data-filter");
        cur = -1; closeConfirm(); refresh();
      });
    });
    var sh = $("#show_hidden");
    if (sh) sh.addEventListener("change", function () {
      showHidden = sh.checked; cur = -1; refresh();
      var url = new URL(location.href);
      if (showHidden) url.searchParams.set("show_hidden", "true"); else url.searchParams.delete("show_hidden");
      history.replaceState({}, "", url);
    });

    // mark all read
    var allBtn = $("[data-act=allread]"), confirmBox = $("#allread-confirm");
    function closeConfirm() { if (confirmBox) { confirmBox.hidden = true; allBtn.hidden = false; } }
    if (allBtn && confirmBox) {
      allBtn.addEventListener("click", function () {
        var n = visibleRows().filter(function (r) { return r.getAttribute("data-read") !== "1"; }).length;
        $(".q", confirmBox).textContent = "Mark " + n + " as read?";
        allBtn.hidden = true; confirmBox.hidden = false;
        $("[data-act=allyes]", confirmBox).focus();
      });
      $("[data-act=allno]", confirmBox).addEventListener("click", closeConfirm);
      $("[data-act=allyes]", confirmBox).addEventListener("click", function () {
        var target = visibleRows().filter(function (r) { return r.getAttribute("data-read") !== "1"; });
        var uids = target.map(function (r) { return r.getAttribute("data-uid"); });
        closeConfirm();
        post("/articles/mark", { uid: uids, read: "true" }).then(function () {
          target.forEach(function (r) { r.classList.add("removed"); r.hidden = true; });
          cur = -1; refresh();
          toast(uids.length + " marked read", function () {
            post("/articles/mark", { uid: uids, read: "false" }).then(function () {
              target.forEach(function (r) { r.classList.remove("removed", "gone"); });
              refresh();
            }).catch(fail);
          });
        }).catch(fail);
      });
    }

    function markDone(r) {
      var uid = r.getAttribute("data-uid");
      var title = $(".article-link", r).textContent.trim();
      r.classList.add("gone");
      post("/article/" + encodeURIComponent(uid) + "/mark", { read: "true" }).then(function () {
        setTimeout(function () {
          r.classList.add("removed"); r.hidden = true; refresh();
          toast("Read: " + (title.length > 30 ? title.slice(0, 30) + "…" : title), function () {
            post("/article/" + encodeURIComponent(uid) + "/mark", { read: "false" }).then(function () {
              r.classList.remove("removed", "gone"); refresh();
            }).catch(fail);
          });
        }, 160);
      }).catch(function () { r.classList.remove("gone"); fail(); });
    }

    function toggleFav(r, quiet) {
      var uid = r.getAttribute("data-uid");
      post("/article/" + encodeURIComponent(uid) + "/favorite", {}).then(function (res) {
        var on = !!res.favorite;
        r.setAttribute("data-fav", on ? "1" : "0");
        var b = $("[data-act=fav]", r);
        if (b) { b.textContent = on ? "(-fav)" : "(+fav)"; b.classList.toggle("on", on); }
        var star = $(".star", r); if (star) star.hidden = !on;
        if (page === "favorites" && !on) { r.classList.add("removed"); r.hidden = true; }
        refresh();
        if (!quiet) toast(on ? "Added to favorites" : "Removed from favorites", function () { toggleFav(r, true); });
      }).catch(fail);
    }

    list.addEventListener("click", function (e) {
      var t = e.target.closest("[data-act]");
      var r = e.target.closest(".article-item");
      if (!r) return;
      cur = visibleRows().indexOf(r); paintCur(false);
      if (!t) return;
      var act = t.getAttribute("data-act");
      if (act === "done") { e.preventDefault(); markDone(r); }
      else if (act === "fav") { e.preventDefault(); toggleFav(r); }
      else if (act === "open" || act === "title") { r.classList.add("seen"); }
    });

    keyHandlers.push(function (e) {
      var vis = visibleRows();
      var k = e.key;
      if (k === "j") { cur = Math.min(vis.length - 1, cur + 1); paintCur(true); return true; }
      if (k === "k") { cur = Math.max(0, cur - 1); paintCur(true); return true; }
      var r = vis[cur];
      if (!r) return false;
      if (k === "Enter") { var rd = $("[data-act=read]", r) || $(".article-link", r); rd.click(); return true; }
      if (k === "o") { $(".article-link", r).click(); return true; }
      if (k === "f") { toggleFav(r); return true; }
      if (k === "e" && $("[data-act=done]", r)) { markDone(r); return true; }
      return false;
    });

    refresh();
  }

  // =====================================================================
  // Feed settings panel (on /feed/:id and /feed)
  // =====================================================================
  $$("[data-feed-panel]").forEach(function (panel) {
    var id = panel.getAttribute("data-feed-panel");
    function save(msg) {
      var data = {};
      ["hide_unread", "enable_readability", "highlight"].forEach(function (n) {
        var el = $("[name=" + n + "]", panel);
        if (el && el.checked) data[n] = "true";
      });
      var cat = $("[name=category]", panel);
      if (cat) { data.category = cat.value; data.set_category = "true"; }
      post("/feed/" + id + "/update", data).then(function () {
        toast(msg || "Saved");
        var item = panel.closest(".feed-item");
        if (item) {
          if (cat) item.setAttribute("data-category", cat.value);
          var hid = $(".is-hidden", item); var hu = $("[name=hide_unread]", panel);
          if (hid && hu) hid.hidden = !hu.checked;
        }
      }).catch(fail);
    }
    $$("input[type=checkbox], select", panel).forEach(function (el) {
      el.addEventListener("change", function () { save(el.tagName === "SELECT" ? "Category saved" : "Saved"); });
    });
    var noscript = $(".panel-save", panel); if (noscript) noscript.hidden = true;
  });

  $$("[data-toggle-panel]").forEach(function (b) {
    b.hidden = false;
    var panel = document.getElementById(b.getAttribute("data-toggle-panel"));
    if (panel) panel.hidden = true;
    b.addEventListener("click", function () {
      if (!panel) return;
      panel.hidden = !panel.hidden;
      b.textContent = panel.hidden ? "(+edit)" : "(-edit)";
      b.classList.toggle("on", !panel.hidden);
    });
  });

  $$("[data-refresh]").forEach(function (b) {
    b.addEventListener("click", function (e) {
      e.preventDefault();
      var id = b.getAttribute("data-refresh"), label = b.textContent;
      b.textContent = b.classList.contains("btn") ? "Refreshing…" : "(refreshing…)";
      b.disabled = true;
      post("/feed/" + id + "/refresh", {}).then(function () {
        if (b.hasAttribute("data-reload")) { location.reload(); return; }
        b.textContent = label; b.disabled = false; toast("Refreshed");
      }).catch(function () { b.textContent = label; b.disabled = false; fail(); });
    });
  });

  // two-step destructive buttons: <button data-confirm="Remove?"> inside a form
  $$("[data-confirm]").forEach(function (b) {
    var armed = false, label = b.textContent, timer;
    b.addEventListener("click", function (e) {
      if (armed) return;
      e.preventDefault();
      armed = true; b.textContent = b.getAttribute("data-confirm"); b.classList.add("danger-t");
      timer = setTimeout(function () { armed = false; b.textContent = label; b.classList.remove("danger-t"); }, 4000);
    });
  });

  // ---- feeds page category chips ----
  var feedChips = $("[data-feed-chips]");
  if (feedChips) {
    var switchCategory = function (category, updateURL) {
      var n = 0;
      $$("[data-fcat]").forEach(function (c) { c.classList.toggle("active", c.getAttribute("data-fcat") === category); });
      $$(".feed-item").forEach(function (f) {
        var fc = f.getAttribute("data-category") || "";
        var vis = category === "all" || (category === "inbox" ? fc === "" : fc === category);
        f.hidden = !vis; if (vis) n++;
      });
      var empty = $("#feeds-empty"); if (empty) empty.hidden = n !== 0;
      if (updateURL) {
        var url = new URL(location.href);
        if (category === "all") url.searchParams.delete("category"); else url.searchParams.set("category", category);
        history.replaceState({}, "", url);
      }
    };
    $$("[data-fcat]").forEach(function (c) {
      c.addEventListener("click", function () { switchCategory(c.getAttribute("data-fcat"), true); });
    });
    switchCategory(feedChips.getAttribute("data-current") || "all", false);
  }

  var opml = $("#upfile");
  if (opml) {
    $("[data-act=import]").addEventListener("click", function () { opml.click(); });
    opml.addEventListener("change", function () { if (opml.files.length) { toast("Importing…"); opml.form.submit(); } });
  }

  // =====================================================================
  // Reader
  // =====================================================================
  var reader = $(".reader");
  if (reader) {
    var bar = $(".progress");
    window.addEventListener("scroll", function () {
      var h = document.documentElement.scrollHeight - innerHeight;
      if (bar) bar.style.width = (h > 0 ? Math.min(100, (scrollY / h) * 100) : 0) + "%";
    }, { passive: true });
    var favBtn = $("[data-reader-fav]");
    var setFav = function (on) {
      favBtn.classList.toggle("on", on);
      $(".ic", favBtn).textContent = on ? "★" : "☆";
      $(".txt", favBtn).textContent = on ? "Faved" : "Fav";
    };
    var flipFav = function (quiet) {
      post("/article/" + encodeURIComponent(favBtn.getAttribute("data-reader-fav")) + "/favorite", {}).then(function (res) {
        setFav(!!res.favorite);
        if (!quiet) toast(res.favorite ? "Added to favorites" : "Removed from favorites", function () { flipFav(true); });
      }).catch(fail);
    };
    if (favBtn) favBtn.addEventListener("click", function (e) { e.preventDefault(); flipFav(false); });
    keyHandlers.push(function (e) {
      var go = function (sel) { var a = $(sel); if (a) { location.href = a.href; return true; } return false; };
      if (e.key === "j") return go("[data-next]");
      if (e.key === "k") return go("[data-prev]");
      if (e.key === "Escape") return go("[data-back]");
      if (e.key === "o") { var o = $("[data-orig]"); if (o) { window.open(o.href, "_blank", "noopener"); return true; } }
      if (e.key === "f" && favBtn) { flipFav(false); return true; }
      return false;
    });
  }

  // =====================================================================
  // Explore / Stream
  // =====================================================================
  var xq = $("#explore-q");
  if (xq) keyHandlers.push(function (e) { if (e.key === "/") { xq.focus(); return true; } return false; });
  $$("form.explore-add").forEach(function (form) {
    form.addEventListener("submit", function () {
      var b = $("button", form);
      b.disabled = true; b.textContent = "Adding…";
    });
  });

  $$("[data-sg]").forEach(function (chip) {
    chip.addEventListener("click", function () {
      var i = chip.getAttribute("data-sg");
      $$("[data-sg]").forEach(function (c) { c.classList.toggle("active", c === chip); });
      $$("[data-sgroup]").forEach(function (g) { g.hidden = g.getAttribute("data-sgroup") !== i; });
    });
  });

  // =====================================================================
  // Preferences
  // =====================================================================
  var prefs = $("#prefs");
  if (prefs) {
    var bar2 = $(".savebar", prefs), dirty = false;
    var markDirty = function () {
      if (dirty) return;
      dirty = true;
      bar2.classList.add("dirty");
      $(".msg", bar2).textContent = "Unsaved changes";
      $("[data-act=discard]", bar2).hidden = false;
    };
    prefs.addEventListener("input", function (e) { if (e.target.name) markDirty(); });
    prefs.addEventListener("change", function (e) { if (e.target.name) markDirty(); });
    $("[data-act=discard]", bar2).addEventListener("click", function () { location.reload(); });
    prefs.addEventListener("submit", function () { dirty = false; });
    window.addEventListener("beforeunload", function (e) { if (dirty) { e.preventDefault(); e.returnValue = ""; } });

    $$("[data-dep]", prefs).forEach(function (el) {
      var src = document.getElementById(el.getAttribute("data-dep"));
      var sync = function () { el.classList.toggle("off", !src.checked); };
      src.addEventListener("change", sync); sync();
    });
    $$("[data-reveal]", prefs).forEach(function (b) {
      b.hidden = false;
      b.addEventListener("click", function () {
        var el = document.getElementById(b.getAttribute("data-reveal"));
        el.type = el.type === "password" ? "text" : "password";
        b.textContent = el.type === "password" ? "show" : "hide";
      });
    });
    $$("[data-clean]", prefs).forEach(function (b) {
      b.addEventListener("click", function () {
        var box = document.getElementById("clean-" + b.getAttribute("data-clean"));
        $$("[data-clean]", prefs).forEach(function (x) { x.hidden = true; });
        box.hidden = false;
      });
    });
    $$("[data-act=cleanno]", prefs).forEach(function (b) {
      b.addEventListener("click", function () {
        b.closest(".confirm").hidden = true;
        $$("[data-clean]", prefs).forEach(function (x) { x.hidden = false; });
      });
    });
  }
})();
