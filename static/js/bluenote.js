document.addEventListener("DOMContentLoaded", async function () {
  if (document.body.id !== "collection") {
    return;
  }

  const wrapper = document.querySelector("section#wrapper");

  if (!wrapper) {
    return;
  }

  /*
   * =========================================================
   * Utilities
   * =========================================================
   */

  function dateKey(date) {
    return (
      date.getFullYear() + "-" +
      String(date.getMonth() + 1).padStart(2, "0") + "-" +
      String(date.getDate()).padStart(2, "0")
    );
  }

  function formatJapaneseDate(date) {
    return (
      date.getFullYear() + "年" +
      (date.getMonth() + 1) + "月" +
      date.getDate() + "日"
    );
  }

  /*
   * =========================================================
   * Load blue note. index
   * =========================================================
   */

  let index;

  try {
    const response = await fetch(
      "/local/blue-note-index.json",
      {
        cache: "no-store",
        credentials: "same-origin"
      }
    );

    if (!response.ok) {
      throw new Error(
        "Index fetch failed: HTTP " + response.status
      );
    }

    index = await response.json();
  } catch (error) {
    console.error(
      "blue-note: index loading failed",
      error
    );

    return;
  }

  /*
   * Convert dates from JSON strings to Date objects.
   */

  const posts = (index.posts || [])
    .map(function (post) {
      const date = new Date(post.created);

      if (Number.isNaN(date.getTime())) {
        return null;
      }

      return {
        id: post.id,
        slug: post.slug,
        title: post.title,
        href: new URL(
          post.url,
          window.location.origin
        ).href,
        date: date,
        updated: post.updated,
        tags: post.tags || []
      };
    })
    .filter(Boolean)
    .sort(function (a, b) {
      return b.date - a.date;
    });

  const tags = Array.isArray(index.tags)
    ? index.tags
    : [];

  /*
   * No public posts = nothing to build.
   */

  if (posts.length === 0) {
    return;
  }

  /*
   * =========================================================
   * Layout
   * =========================================================
   */

  const layout = document.createElement("div");
  layout.className = "blue-note-layout";

  const main = document.createElement("main");
  main.className = "blue-note-main";

  const sidebar = document.createElement("aside");
  sidebar.className = "blue-note-sidebar";

  wrapper.parentNode.insertBefore(layout, wrapper);

  layout.appendChild(main);
  layout.appendChild(sidebar);

  main.appendChild(wrapper);

  /*
   * Sidebar panel utility
   */

  function createPanel(title) {
    const panel = document.createElement("section");
    panel.className = "blue-note-panel";

    if (title) {
      const heading = document.createElement("h2");
      heading.textContent = title;
      panel.appendChild(heading);
    }

    sidebar.appendChild(panel);

    return panel;
  }

  /*
   * =========================================================
   * Calendar
   * =========================================================
   */

  const postsByDate = new Map();

  posts.forEach(function (post) {
    const key = dateKey(post.date);

    if (!postsByDate.has(key)) {
      postsByDate.set(key, []);
    }

    postsByDate.get(key).push(post);
  });

  const calendarPanel = createPanel("");
  calendarPanel.classList.add(
    "blue-note-calendar-panel"
  );

  /*
   * Initial month = newest post
   */

  let currentDate = new Date(
    posts[0].date.getFullYear(),
    posts[0].date.getMonth(),
    1
  );

  function renderCalendar() {
    calendarPanel.innerHTML = "";

    const year = currentDate.getFullYear();
    const month = currentDate.getMonth();

    /*
     * Header
     */

    const header = document.createElement("div");
    header.className =
      "blue-note-calendar-header";

    const prev = document.createElement("button");
    prev.type = "button";
    prev.textContent = "‹";
    prev.setAttribute(
      "aria-label",
      "Previous month"
    );

    const title = document.createElement("span");
    title.className =
      "blue-note-calendar-title";

    title.textContent =
      year + " / " +
      String(month + 1).padStart(2, "0");

    const next = document.createElement("button");
    next.type = "button";
    next.textContent = "›";
    next.setAttribute(
      "aria-label",
      "Next month"
    );

    header.appendChild(prev);
    header.appendChild(title);
    header.appendChild(next);

    calendarPanel.appendChild(header);

    /*
     * Calendar table
     */

    const table = document.createElement("table");
    table.className = "blue-note-calendar";

    const weekdays = [
      "Sun",
      "Mon",
      "Tue",
      "Wed",
      "Thu",
      "Fri",
      "Sat"
    ];

    const thead = document.createElement("thead");
    const headRow = document.createElement("tr");

    weekdays.forEach(function (weekday) {
      const th = document.createElement("th");
      th.textContent = weekday;
      headRow.appendChild(th);
    });

    thead.appendChild(headRow);
    table.appendChild(thead);

    const tbody = document.createElement("tbody");

    const firstDay =
      new Date(year, month, 1).getDay();

    const daysInMonth =
      new Date(year, month + 1, 0).getDate();

    let day = 1;

    for (let week = 0; week < 6; week++) {
      const row = document.createElement("tr");

      for (
        let weekday = 0;
        weekday < 7;
        weekday++
      ) {
        const cell =
          document.createElement("td");

        if (
          (week === 0 && weekday < firstDay) ||
          day > daysInMonth
        ) {
          cell.className = "empty";
        } else {
          const cellDate =
            new Date(year, month, day);

          const key = dateKey(cellDate);
          const dayPosts =
            postsByDate.get(key);

          if (
            dayPosts &&
            dayPosts.length > 0
          ) {
            const link =
              document.createElement("a");

            link.textContent = day;

            /*
             * If multiple posts exist on one day,
             * the day links to the newest one.
             */

            link.href = dayPosts[0].href;

            link.title = dayPosts
              .map(function (post) {
                return post.title;
              })
              .join("\n");

            cell.className = "has-post";
            cell.appendChild(link);
          } else {
            cell.textContent = day;
          }

          day++;
        }

        row.appendChild(cell);
      }

      tbody.appendChild(row);

      if (day > daysInMonth) {
        break;
      }
    }

    table.appendChild(tbody);
    calendarPanel.appendChild(table);

    /*
     * Month navigation
     */

    prev.addEventListener(
      "click",
      function () {
        currentDate =
          new Date(year, month - 1, 1);

        renderCalendar();
      }
    );

    next.addEventListener(
      "click",
      function () {
        currentDate =
          new Date(year, month + 1, 1);

        renderCalendar();
      }
    );
  }

  renderCalendar();

  /*
   * =========================================================
   * Recent Posts
   * =========================================================
   */

  const recentPanel =
    createPanel("Recent Posts");

  const recentList =
    document.createElement("ul");

  recentList.className = "blue-note-recent";

  posts.slice(0, 5).forEach(function (post) {
    const item =
      document.createElement("li");

    const link =
      document.createElement("a");

    link.href = post.href;
    link.textContent = post.title;

    const date =
      document.createElement("span");

    date.className =
      "blue-note-recent-date";

    date.textContent =
      formatJapaneseDate(post.date);

    item.appendChild(link);
    item.appendChild(date);

    recentList.appendChild(item);
  });

  recentPanel.appendChild(recentList);

  /*
   * =========================================================
   * Tags
   * =========================================================
   */

  const tagPanel = createPanel("Tags");

  const tagCloud =
    document.createElement("div");

  tagCloud.className =
    "blue-note-tag-cloud";

  tags.forEach(function (tag) {
    const link =
      document.createElement("a");

    link.className =
      "blue-note-sidebar-tag";

    link.href = tag.url;
    link.textContent = "#" + tag.name;

    link.title =
      tag.count +
      (tag.count === 1
        ? " post"
        : " posts");

    tagCloud.appendChild(link);
  });

  tagPanel.appendChild(tagCloud);

  /*
   * =========================================================
   * Links
   * =========================================================
   */

  const linksPanel = createPanel("Links");
  const linksList = document.createElement("div");
  linksList.className = "blue-note-links";

  linksPanel.appendChild(linksList);

  fetch("/local/blue-note-links.json", {
    cache: "no-store"
  })
    .then(function (response) {
      if (!response.ok) {
        throw new Error(
          "Failed to load links: HTTP " + response.status
        );
      }

      return response.json();
    })
    .then(function (links) {
      links.forEach(function (site) {
        if (!site.name || !site.url) {
          return;
        }

        const item = document.createElement("div");
        item.className = "blue-note-link-item";

        const link = document.createElement("a");
        link.href = site.url;
        link.target = "_blank";
        link.rel = "noopener noreferrer";
        link.title = site.name;

        if (site.banner) {
          const banner = document.createElement("img");

          banner.src = site.banner;
          banner.alt = site.name;
          banner.className = "blue-note-link-banner";

          /*
           * If the banner cannot be loaded,
           * fall back to the site name.
           */
          banner.addEventListener("error", function () {
            link.replaceChildren();

            const name = document.createElement("span");
            name.className = "blue-note-link-name";
            name.textContent = site.name;

            link.appendChild(name);
          });

          link.appendChild(banner);
        } else {
          const name = document.createElement("span");
          name.className = "blue-note-link-name";
          name.textContent = site.name;

          link.appendChild(name);
        }

        item.appendChild(link);
        linksList.appendChild(item);
      });

      console.log(
        "blue-note links loaded:",
        links.length,
        "links"
      );
    })
    .catch(function (error) {
      console.error("blue-note links error:", error);
    });

/*
 * Link exchange notice
 */
const linkNotice = document.createElement("div");
linkNotice.className = "blue-note-link-notice";

const freeText = document.createElement("div");
freeText.textContent = "リンクフリー";

const welcomeText = document.createElement("div");
welcomeText.textContent = "相互リンク歓迎";

const banner = document.createElement("img");
banner.src = "/local/banners/bluenote_bn.png";
banner.alt = "blue note.";
banner.className = "blue-note-own-banner";

linkNotice.appendChild(freeText);
linkNotice.appendChild(welcomeText);
linkNotice.appendChild(banner);

linksPanel.appendChild(linkNotice);

  /*
   * =========================================================
   * Debug
   * =========================================================
   */

  console.log(
    "blue-note.js initialized:",
    posts.length,
    "posts,",
    tags.length,
    "tags"
  );
});

/*
 * =========================================================
 * Article Share
 * =========================================================
 */

document.addEventListener("DOMContentLoaded", function () {
  if (document.body.id !== "post") {
    return;
  }

  const article = document.querySelector("article#post-body");

  if (!article) {
    return;
  }

  const titleElement = article.querySelector("#title");
  const articleTitle = titleElement
    ? titleElement.textContent.trim()
    : document.title;

  const articleURL = window.location.href;

  /*
   * Share container
   */
  const share = document.createElement("section");
  share.className = "blue-note-share";

  const heading = document.createElement("div");
  heading.className = "blue-note-share-title";
  heading.textContent = "Share";

  const buttons = document.createElement("div");
  buttons.className = "blue-note-share-buttons";

  share.appendChild(heading);
  share.appendChild(buttons);

  /*
   * Mastodon
   */
  const mastodon = document.createElement("a");
  mastodon.className =
    "blue-note-share-button blue-note-share-mastodon";

  mastodon.href =
    "https://share.joinmastodon.org/?text=" +
    encodeURIComponent(articleTitle + "\n" + articleURL);

  mastodon.target = "_blank";
  mastodon.rel = "noopener noreferrer";
  mastodon.title = "Share on Mastodon";
  mastodon.setAttribute("aria-label", "Share on Mastodon");

  const mastodonIcon = document.createElement("img");
  mastodonIcon.src = "/local/icons/mastodon.svg";
  mastodonIcon.alt = "";

  mastodon.appendChild(mastodonIcon);
  buttons.appendChild(mastodon);

  /*
   * Misskey
   *
   * Misskey Hub forwards the request to the user's instance.
   */
  const misskey = document.createElement("a");
  misskey.className =
    "blue-note-share-button blue-note-share-misskey";

  misskey.href =
    "https://misskey-hub.net/share/?" +
    "text=" + encodeURIComponent(articleTitle) +
    "&url=" + encodeURIComponent(articleURL);

  misskey.target = "_blank";
  misskey.rel = "noopener noreferrer";
  misskey.title = "Share on Misskey";
  misskey.setAttribute("aria-label", "Share on Misskey");

  const misskeyIcon = document.createElement("img");
  misskeyIcon.src = "/local/icons/misskey.svg";
  misskeyIcon.alt = "";

  misskey.appendChild(misskeyIcon);
  buttons.appendChild(misskey);

  /*
   * X official share button
   */
  const xWrapper = document.createElement("span");
  xWrapper.className = "blue-note-share-x";

  const xLink = document.createElement("a");
  xLink.href = "https://x.com/share?ref_src=twsrc%5Etfw";
  xLink.className = "twitter-share-button";
  xLink.setAttribute("data-text", articleTitle);
  xLink.setAttribute("data-url", articleURL);
  xLink.setAttribute("data-show-count", "false");
  xLink.textContent = "Post";

  xWrapper.appendChild(xLink);
  buttons.appendChild(xWrapper);

    /*
     * Copy URL
     */
    const copyButton = document.createElement("button");
    copyButton.type = "button";
    copyButton.className =
      "blue-note-share-button blue-note-share-copy";

    copyButton.title = "Copy link";
    copyButton.setAttribute("aria-label", "Copy link");

    const copyIcon = document.createElement("img");
    copyIcon.src = "/local/icons/copy.svg";
    copyIcon.alt = "";

    copyButton.appendChild(copyIcon);
    buttons.appendChild(copyButton);

    copyButton.addEventListener("click", async function () {
      try {
        await navigator.clipboard.writeText(articleURL);

        copyButton.classList.add("copied");
        copyButton.title = "Copied!";
        copyButton.setAttribute("aria-label", "Copied!");

        setTimeout(function () {
          copyButton.classList.remove("copied");
          copyButton.title = "Copy link";
          copyButton.setAttribute("aria-label", "Copy link");
        }, 1500);
      } catch (error) {
        console.error("blue-note: clipboard copy failed", error);
      }
    });


  /*
   * Insert after article
   */
  article.insertAdjacentElement("afterend", share);

  /*
   * Load X widget
   */
  if (!document.querySelector('script[src*="platform.x.com/widgets.js"]')) {
    const script = document.createElement("script");
    script.src = "https://platform.x.com/widgets.js";
    script.async = true;
    script.charset = "utf-8";

    document.body.appendChild(script);
  }

  console.log("blue-note article share initialized");
});

