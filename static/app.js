// 1. Επιλογή των βασικών στοιχείων από το HTML (DOM)
const authContainer = document.getElementById("auth-container");
const forumApp = document.getElementById("forum-app");
const logBox = document.getElementById("log");
const regBox = document.getElementById("reg");

const showRegisterBtn = document.getElementById("show-register");
const showLoginBtn = document.getElementById("show-login");
const registerForm = document.getElementById("register-form");
const loginForm = document.getElementById("login-form");
const logoutBtn = document.getElementById("logout-button");
let currentUser = null;
let chatSocket = null;
let selectedUserId = null;
let oldestMessageId = null;
let loadingOlderMessages = false;
let chatAtBottom = true;
let typingStopTimer = null;

// 2. Εναλλαγή οθόνης Login <-> Register
showRegisterBtn.addEventListener("click", () => {
    logBox.style.display = "none";
    regBox.style.display = "block";
});

showLoginBtn.addEventListener("click", () => {
    regBox.style.display = "none";
    logBox.style.display = "block";
});

// 3. Βοηθητικές συναρτήσεις εμφάνισης Login ή Forum
function showForum() {
    authContainer.style.display = "none";
    forumApp.style.display = "block";
}

function showAuth() {
    forumApp.style.display = "none";
    authContainer.style.display = "grid";
}

// 4. Έλεγχος αν ο χρήστης είναι ήδη συνδεδεμένος (π.χ. μετά από F5 refresh)
async function checkSession() {
    const res = await fetch("/api/me");
    if (res.ok) {
        currentUser = await res.json();
        showForum();
        await loadPosts();
        await loadUsers();
        connectChat();
    } else {
        showAuth();
    }
}

// 5. Αποστολή φόρμας Register -> Go Backend (/api/register)
registerForm.addEventListener("submit", async (e) => {
    e.preventDefault(); // Σταματάει το refresh της σελίδας

    const payload = {
        nickname: document.getElementById("register-nickname").value.trim(),
        age: Number(document.getElementById("register-age").value),
        gender: document.getElementById("register-gender").value,
        first_name: document.getElementById("register-first-name").value.trim(),
        last_name: document.getElementById("register-last-name").value.trim(),
        email: document.getElementById("register-email").value.trim(),
        password: document.getElementById("register-password").value
    };

    const res = await fetch("/api/register", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload)
    });

    if (res.ok) {
        alert("Account created! You can now log in.");
        registerForm.reset();
        regBox.style.display = "none";
        logBox.style.display = "block";
    } else {
        const errorText = await res.text();
        alert("Register failed: " + errorText);
    }
});

// 6. Αποστολή φόρμας Login -> Go Backend (/api/login)
loginForm.addEventListener("submit", async (e) => {
    e.preventDefault();

    const payload = {
        identifier: document.getElementById("login-username").value.trim(),
        password: document.getElementById("login-password").value
    };

    const res = await fetch("/api/login", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload)
    });

    if (res.ok) {
        loginForm.reset();
        await checkSession();
    } else {
        const errorText = await res.text();
        alert("Login failed: " + errorText);
    }
});

// 7. Logout -> Go Backend (/api/logout)
logoutBtn.addEventListener("click", async () => {
    const res = await fetch("/api/logout", { method: "POST" });
    if (res.ok) {
        currentUser = null;
        if (chatSocket) chatSocket.close();
        chatSocket = null;
        showAuth();
    }
});

const usersList = document.getElementById("users-list");
const onlineCount = document.getElementById("online-count");
const chatHeaderName = document.getElementById("chat-user-name");
const chatHeaderStatus = document.getElementById("chat-user-status");
const chatMessages = document.getElementById("chat-messages");
const messageForm = document.getElementById("message-form");
const messageInput = document.getElementById("message-input");

async function loadUsers() {
    const res = await fetch("/api/users");
    if (!res.ok) return;
    const users = await res.json();
    let online = 0;
    usersList.innerHTML = "";
    users.forEach(user => {
        if (user.online) online++;
        const button = document.createElement("button");
        button.type = "button";
        button.className = "chat-user-button";
        if (user.id === selectedUserId) button.classList.add("active");
        button.innerHTML = `
            <span class="chat-user-info">
                <strong>${escapeHtml(user.username)}</strong>
                <small>${user.last_at ? "Recent conversation" : "No messages yet"}</small>
            </span>
            <span class="chat-user-badges">
                ${user.unread > 0 ? `<span class="unread-badge">${user.unread > 99 ? "99+" : user.unread}</span>` : ""}
                <span class="chat-user-status ${user.online ? "online" : "offline"}" title="${user.online ? "Online" : "Offline"}"></span>
            </span>
        `;
        button.addEventListener("click", () => openConversation(user));
        usersList.appendChild(button);
    });
    onlineCount.textContent = `${online} online`;
}

async function openConversation(user) {
    selectedUserId = user.id;
    oldestMessageId = null;
    chatHeaderName.textContent = `@${user.username}`;
    chatHeaderStatus.textContent = user.online ? "Online" : "Offline";
    chatHeaderStatus.dataset.online = String(user.online);
    chatMessages.innerHTML = "";
    await loadOlderMessages(true);
    await markConversationRead(user.id);
    await loadUsers();
}

async function markConversationRead(contactId) {
    await fetch("/api/messages/read", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ contact_id: contactId })
    });
}

async function loadOlderMessages(initial = false) {
    if (!selectedUserId || loadingOlderMessages) return;
    loadingOlderMessages = true;
    const previousScrollHeight = chatMessages.scrollHeight;
    const before = initial ? "" : `&before_id=${oldestMessageId}`;
    const res = await fetch(`/api/messages?user_id=${selectedUserId}${before}`);
    if (res.ok) {
        const messages = await res.json();
        if (messages.length > 0) {
            oldestMessageId = messages[0].id;
            const html = messages.map(messageMarkup).join("");
            if (initial) {
                chatMessages.innerHTML = html;
                chatMessages.scrollTop = chatMessages.scrollHeight;
                chatAtBottom = true;
            } else {
                chatMessages.insertAdjacentHTML("afterbegin", html);
                chatMessages.scrollTop = chatMessages.scrollHeight - previousScrollHeight;
            }
        } else if (initial) {
            chatMessages.innerHTML = `<div class="chat-empty">No messages yet.</div>`;
        }
    }
    loadingOlderMessages = false;
}

function messageMarkup(message) {
    const mine = message.sender_id === currentUser.id;
    const date = new Date(message.created_at);
    const time = Number.isNaN(date.getTime()) ? "" : date.toLocaleString();
    return `
        <div class="chat-message ${mine ? "mine" : ""}" data-message-id="${message.id}">
            <div class="chat-message-meta">
                <strong>@${escapeHtml(message.author)}</strong>
                <time>${escapeHtml(time)}</time>
            </div>
            <div class="chat-message-content">${escapeHtml(message.content)}</div>
        </div>
    `;
}

function connectChat() {
    if (chatSocket) chatSocket.close();
    const protocol = location.protocol === "https:" ? "wss:" : "ws:";
    chatSocket = new WebSocket(`${protocol}//${location.host}/ws`);
    chatSocket.addEventListener("message", event => {
        const payload = JSON.parse(event.data);
        if (payload.type === "presence") {
            loadUsers();
        } else if (payload.type === "typing") {
            if (payload.sender_id === selectedUserId) {
                chatHeaderStatus.textContent = payload.typing ? "typing..." : (
                    chatHeaderStatus.dataset.online === "true" ? "Online" : "Offline"
                );
            }
        } else if (payload.type === "message") {
            const message = payload.message;
            if (selectedUserId && (message.sender_id === selectedUserId || message.receiver_id === selectedUserId)) {
                chatMessages.querySelector(".chat-empty")?.remove();
                chatMessages.insertAdjacentHTML("beforeend", messageMarkup(message));
                if (chatAtBottom) chatMessages.scrollTop = chatMessages.scrollHeight;
                if (message.sender_id === selectedUserId) markConversationRead(selectedUserId);
            }
            loadUsers();
        }
    });
}

let chatScrollTimer = null;
chatMessages.addEventListener("scroll", () => {
    const distanceFromBottom = chatMessages.scrollHeight - chatMessages.scrollTop - chatMessages.clientHeight;
    chatAtBottom = distanceFromBottom < 60;
    if (chatMessages.scrollTop > 100) return;
    clearTimeout(chatScrollTimer);
    chatScrollTimer = setTimeout(() => loadOlderMessages(false), 180);
});

messageInput.addEventListener("input", () => {
    if (!selectedUserId || !chatSocket || chatSocket.readyState !== WebSocket.OPEN) return;
    chatSocket.send(JSON.stringify({
        type: "typing",
        receiver_id: selectedUserId,
        typing: true
    }));
    clearTimeout(typingStopTimer);
    typingStopTimer = setTimeout(() => {
        chatSocket.send(JSON.stringify({
            type: "typing",
            receiver_id: selectedUserId,
            typing: false
        }));
    }, 1200);
});

messageForm.addEventListener("submit", async event => {
    event.preventDefault();
    if (!selectedUserId) return;
    const content = messageInput.value.trim();
    if (!content) return;
    const res = await fetch("/api/messages/create", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ receiver_id: selectedUserId, content })
    });
    if (res.ok) {
        messageInput.value = "";
        clearTimeout(typingStopTimer);
        if (chatSocket && chatSocket.readyState === WebSocket.OPEN) {
            chatSocket.send(JSON.stringify({
                type: "typing",
                receiver_id: selectedUserId,
                typing: false
            }));
        }
    }
});

const createPostForm = document.getElementById("create-post-form");
const postsList = document.getElementById("posts-list");

function escapeHtml(value) {
    return String(value ?? "")
        .replaceAll("&", "&amp;")
        .replaceAll("<", "&lt;")
        .replaceAll(">", "&gt;")
        .replaceAll('"', "&quot;")
        .replaceAll("'", "&#039;");
}

// Φόρτωση όλων των posts από το Go Backend
async function loadPosts() {
    const res = await fetch("/api/posts");
    if (!res.ok) return;
    const posts = await res.json();

    postsList.innerHTML = "";
    posts.forEach(post => {
        const needsReadMore = post.content.length > 220;
        const card = document.createElement("article");
        card.className = "post-card";
        card.style.cssText = "padding:1.25rem; border:1px solid rgba(255,255,255,0.09); border-radius:16px; background:rgba(255,255,255,0.035);";

        card.innerHTML = `
            <div style="display:flex; justify-content:space-between; color:var(--muted); font-size:0.8rem; margin-bottom:0.5rem;">
                <span>@${escapeHtml(post.author)}</span>
                <span>${escapeHtml(post.categories || "General")}</span>
            </div>
            <h3 style="margin-bottom:0.5rem;">${escapeHtml(post.title)}</h3>
            <p class="post-content" style="color:#dedbd7; line-height:1.6;">${needsReadMore ? escapeHtml(post.content.slice(0, 220)) + "..." : escapeHtml(post.content)}</p>
            ${needsReadMore ? `<button type="button" class="read-more-button">Read more</button>` : ""}
            <div class="post-actions">
                <button type="button" class="like-button ${post.user_reaction === 1 ? "active" : ""}">Like <span class="like-count">${post.likes}</span></button>
                <button type="button" class="dislike-button ${post.user_reaction === -1 ? "active" : ""}">Dislike <span class="dislike-count">${post.dislikes}</span></button>
                <button type="button" class="comments-toggle">Comments</button>
                ${currentUser && post.user_id === currentUser.id ? `<button type="button" class="delete-post-button">Delete</button>` : ""}
            </div>
            <div class="comments-section" style="display:none; margin-top:1rem; padding-top:1rem; border-top:1px solid rgba(255,255,255,0.08);">
                <div class="comments-list" style="display:grid; gap:0.5rem; margin-bottom:0.75rem;"></div>
                <form class="comment-form" style="display:flex; gap:0.5rem;">
                    <input type="text" placeholder="Write a comment..." required style="flex:1; height:38px; padding:0 0.75rem; border-radius:8px; border:1px solid rgba(255,255,255,0.12); background:rgba(0,0,0,0.3); color:var(--text);">
                    <button type="submit" style="padding:0 1rem; height:38px; border:0; border-radius:8px; background:var(--accent); color:#111; font-weight:600; cursor:pointer;">Reply</button>
                </form>
            </div>
        `;

        const contentElement = card.querySelector(".post-content");
        const readMoreButton = card.querySelector(".read-more-button");
        if (readMoreButton) {
            readMoreButton.addEventListener("click", () => {
                const expanded = readMoreButton.textContent === "Read less";
                contentElement.textContent = expanded ? post.content.slice(0, 220) + "..." : post.content;
                readMoreButton.textContent = expanded ? "Read more" : "Read less";
            });
        }

        const commentsSection = card.querySelector(".comments-section");
        card.querySelector(".comments-toggle").addEventListener("click", async () => {
            const isHidden = commentsSection.style.display === "none";
            commentsSection.style.display = isHidden ? "block" : "none";
            if (isHidden) await loadComments(post.id, card.querySelector(".comments-list"));
        });

        async function react(value) {
            const res = await fetch("/api/posts/reaction", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ post_id: post.id, value })
            });
            if (res.ok) await loadPosts();
        }
        card.querySelector(".like-button").addEventListener("click", () => react(1));
        card.querySelector(".dislike-button").addEventListener("click", () => react(-1));

        const deleteButton = card.querySelector(".delete-post-button");
        if (deleteButton) {
            deleteButton.addEventListener("click", async () => {
                if (!confirm("Delete this post?")) return;
                const res = await fetch("/api/posts/delete", {
                    method: "DELETE",
                    headers: { "Content-Type": "application/json" },
                    body: JSON.stringify({ post_id: post.id })
                });
                if (res.ok) await loadPosts();
            });
        }

        // Υποβολή νέου σχολίου
        card.querySelector(".comment-form").addEventListener("submit", async (e) => {
            e.preventDefault();
            const input = e.target.querySelector("input");
            const res = await fetch("/api/comments/create", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ post_id: post.id, content: input.value.trim() })
            });
            if (res.ok) {
                input.value = "";
                await loadComments(post.id, card.querySelector(".comments-list"));
            }
        });

        postsList.appendChild(card);
    });
}

async function loadComments(postId, container) {
    const res = await fetch(`/api/comments?post_id=${postId}`);
    if (!res.ok) return;
    const comments = await res.json();
    container.innerHTML = comments.length
        ? comments.map(c => `<div style="font-size:0.85rem; padding:0.5rem 0.75rem; border-radius:8px; background:rgba(0,0,0,0.25);"><strong>@${escapeHtml(c.author)}:</strong> ${escapeHtml(c.content)}</div>`).join("")
        : `<div style="font-size:0.8rem; color:var(--muted);">No comments yet.</div>`;
}

// Δημιουργία νέου post
createPostForm.addEventListener("submit", async (e) => {
    e.preventDefault();
    const payload = {
        title: document.getElementById("post-title").value.trim(),
        content: document.getElementById("post-content").value.trim(),
        categories: document.getElementById("post-categories").value.trim()
    };
    const res = await fetch("/api/posts/create", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload)
    });
    if (res.ok) {
        createPostForm.reset();
        await loadPosts();
    }
});

checkSession();
