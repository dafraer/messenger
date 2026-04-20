// Reveal Material Symbol icons only after the icon font is loaded
document.fonts.load('1em "Material Symbols Outlined"').then(() => {
    document.documentElement.classList.add('fonts-ready');
});

document.addEventListener('DOMContentLoaded', () => {
    // ── State ──────────────────────────────────────────────────────────────
    let authToken = localStorage.getItem('authToken');
    let currentUsername = localStorage.getItem('currentUsername');
    let webSocket = null;
    let currentChatId = null;
    let chats = {};
    let messages = {};

    // ── DOM refs ───────────────────────────────────────────────────────────
    const authContainer       = document.getElementById('auth-container');
    const appContainer        = document.getElementById('app-container');
    const loginView           = document.getElementById('login-view');
    const registerView        = document.getElementById('register-view');
    const chatsView           = document.getElementById('chats-view');
    const chatView            = document.getElementById('chat-view');

    const loginForm           = document.getElementById('login-form');
    const registerForm        = document.getElementById('register-form');
    const messageForm         = document.getElementById('message-form');

    const loginUsernameInput        = document.getElementById('login-username');
    const loginPasswordInput        = document.getElementById('login-password');
    const registerUsernameInput     = document.getElementById('register-username');
    const registerPasswordInput     = document.getElementById('register-password');
    const registerConfirmPasswordInput = document.getElementById('register-confirm-password');
    const messageInput              = document.getElementById('message-input');
    const searchInput               = document.getElementById('search-username');

    const loggedInUsernameSpan  = document.getElementById('logged-in-username');
    const chatListUl            = document.getElementById('chat-list');
    const messagesContainer     = document.getElementById('messages-container');
    const currentChatNameH2     = document.getElementById('current-chat-name');

    const logoutScreen          = document.getElementById('logout-screen');
    const logoutButton          = document.getElementById('logout-button');
    const mobileLogoutBtn       = document.getElementById('mobile-logout-btn');
    const showRegisterBtn       = document.getElementById('show-register');
    const showLoginBtn          = document.getElementById('show-login');
    const newChatBtn            = document.getElementById('new-chat-btn');
    const mobileNewChatBtn      = document.getElementById('mobile-new-chat-btn');
    const backToChatsBtn        = document.getElementById('back-to-chats');
    const navMessagesBtn        = document.getElementById('nav-messages-btn');

    const loginErrorP      = document.getElementById('login-error');
    const registerErrorP   = document.getElementById('register-error');
    const registerSuccessP = document.getElementById('register-success');
    const chatsErrorP      = document.getElementById('chats-error');
    const messageErrorP    = document.getElementById('message-error');
    const searchErrorP     = document.getElementById('search-error');
    const searchInfoP      = document.getElementById('search-info');

    const API_BASE_URL = window.location.origin;
    const WS_BASE_URL  = API_BASE_URL.replace(/^http/, 'ws');

    // ── Helpers ────────────────────────────────────────────────────────────
    function displayError(el, msg) {
        if (!el) return;
        el.textContent = msg || 'An unexpected error occurred.';
        el.style.display = '';
        console.error(msg);
    }

    function clearError(el) {
        if (!el) return;
        el.textContent = '';
        el.style.display = 'none';
    }

    function displayInfo(el, msg) {
        if (!el) return;
        el.textContent = msg || '';
        el.style.display = '';
    }

    // ── View switching ─────────────────────────────────────────────────────
    function showView(viewToShow) {
        authContainer.style.display = 'none';
        appContainer.style.display  = 'none';
        loginView.style.display     = 'none';
        registerView.style.display  = 'none';

        if (viewToShow === 'login') {
            document.documentElement.classList.remove('is-authed');
            authContainer.style.display = '';
            loginView.style.display     = '';
        } else if (viewToShow === 'register') {
            document.documentElement.classList.remove('is-authed');
            authContainer.style.display = '';
            registerView.style.display  = '';
        } else if (viewToShow === 'app') {
            appContainer.style.display = 'flex';
            showChatsList();
        }
    }

    function showChatsList() {
        chatsView.style.display = 'flex';
        chatView.style.display  = 'none';
    }

    function showChatView() {
        chatsView.style.display = 'none';
        chatView.style.display  = 'flex';
    }

    // ── Last-message persistence ───────────────────────────────────────────
    const LAST_MSG_KEY = `lastMessages_${currentUsername || ''}`;

    function loadStoredPreviews() {
        try { return JSON.parse(localStorage.getItem(LAST_MSG_KEY) || '{}'); } catch { return {}; }
    }

    function persistPreview(chatId, text) {
        const stored = loadStoredPreviews();
        stored[chatId] = text;
        localStorage.setItem(LAST_MSG_KEY, JSON.stringify(stored));
    }

    function clearStoredPreviews() {
        localStorage.removeItem(LAST_MSG_KEY);
    }

    // ── API helper ─────────────────────────────────────────────────────────
    async function makeApiRequest(endpoint, method = 'GET', body = null, requiresAuth = true) {
        const url = `${API_BASE_URL}${endpoint}`;
        const headers = { 'Content-Type': 'application/json' };
        if (requiresAuth && authToken) {
            headers['Authorization'] = `Bearer ${authToken}`;
        }

        const options = { method, headers };
        if (body) options.body = JSON.stringify(body);

        try {
            const response = await fetch(url, options);
            if (!response.ok) {
                let errorMsg = `HTTP error! Status: ${response.status}`;
                try { errorMsg = (await response.text()) || errorMsg; } catch (_) {}
                throw new Error(errorMsg);
            }
            const ct = response.headers.get('content-type');
            return ct && ct.includes('application/json')
                ? await response.json()
                : await response.text();
        } catch (error) {
            console.error(`API Request Error (${method} ${endpoint}):`, error);
            throw error;
        }
    }

    // ── Authentication ─────────────────────────────────────────────────────
    async function handleLogin(event) {
        event.preventDefault();
        clearError(loginErrorP);
        const username = loginUsernameInput.value.trim();
        const password = loginPasswordInput.value.trim();
        if (!username || !password) {
            displayError(loginErrorP, 'Username and password are required.');
            return;
        }
        try {
            const data = await makeApiRequest('/login', 'POST', { username, password }, false);
            if (data && typeof data === 'string') {
                authToken       = data;
                currentUsername = username;
                localStorage.setItem('authToken', authToken);
                localStorage.setItem('currentUsername', currentUsername);
                initializeChatApp();
            } else {
                displayError(loginErrorP, 'Login failed: Invalid response from server.');
            }
        } catch (error) {
            displayError(loginErrorP, 'Something went wrong, please try again.');
        }
    }

    async function handleRegister(event) {
        event.preventDefault();
        clearError(registerErrorP);
        clearError(registerSuccessP);
        const username        = registerUsernameInput.value.trim();
        const password        = registerPasswordInput.value;
        const confirmPassword = registerConfirmPasswordInput.value;
        if (!username || !password || !confirmPassword) {
            displayError(registerErrorP, 'All fields are required.');
            return;
        }
        if (password.length < 8) {
            displayError(registerErrorP, 'Password must be at least 8 characters.');
            return;
        }
        if (password !== confirmPassword) {
            displayError(registerErrorP, 'Passwords do not match.');
            return;
        }
        try {
            await makeApiRequest('/register', 'POST', { username, password }, false);
            registerForm.reset();
            displayInfo(registerSuccessP, 'Registration successful! Please sign in.');
        } catch (error) {
            displayError(registerErrorP, 'Something went wrong, please try again.');
        }
    }

    function handleLogout() {
        // Show logout screen immediately, hide app
        logoutScreen.style.display = 'flex';
        appContainer.style.display = 'none';

        authToken       = null;
        currentUsername = null;
        localStorage.removeItem('authToken');
        localStorage.removeItem('currentUsername');

        if (webSocket) { webSocket.close(); webSocket = null; }

        chats         = {};
        messages      = {};
        currentChatId = null;
        clearStoredPreviews();

        chatListUl.innerHTML = '';
        resetMessagesContainer();
        currentChatNameH2.textContent = '';

        // Transition to login after a brief pause
        setTimeout(() => {
            logoutScreen.style.display = 'none';
            showView('login');
            loginForm.reset();
            registerForm.reset();
            clearError(loginErrorP);
            clearError(registerErrorP);
            clearError(registerSuccessP);
        }, 600);
    }

    function resetMessagesContainer() {
        messagesContainer.innerHTML =
            '<p id="no-chat-selected" class="m-auto text-on-surface-variant text-sm text-center">Select a chat to start messaging.</p>';
    }

    // ── WebSocket ──────────────────────────────────────────────────────────
    function connectWebSocket() {
        if (webSocket && webSocket.readyState === WebSocket.OPEN) return;
        if (!authToken) { handleLogout(); return; }

        const wsUrl = `${WS_BASE_URL}/ws`;
        webSocket = new WebSocket(wsUrl);

        webSocket.onopen = () => console.log('WebSocket connected.');

        webSocket.onmessage = (event) => {
            try {
                const msg = JSON.parse(event.data);
                if (msg.from && msg.chat_id && msg.text) handleIncomingMessage(msg);
            } catch (e) { console.error('WS parse error:', e); }
        };

        webSocket.onerror = (err) => {
            console.error('WebSocket error:', err);
            displayError(messageErrorP, 'WebSocket error. Real-time updates may fail.');
        };

        webSocket.onclose = (event) => {
            console.log('WebSocket closed:', event.code, event.reason);
            webSocket = null;
            if (authToken) {
                displayError(messageErrorP, 'Disconnected. Reconnecting…');
                setTimeout(connectWebSocket, 5000);
            }
        };
    }

    function sendMessage(text) {
        if (!webSocket || webSocket.readyState !== WebSocket.OPEN) {
            displayError(messageErrorP, 'Not connected. Reconnecting…');
            connectWebSocket();
            return;
        }
        if (!currentChatId) { displayError(messageErrorP, 'No active chat selected.'); return; }
        if (!text.trim()) return;

        const msg = { from: currentUsername, chat_id: currentChatId, text: text.trim() };
        try {
            webSocket.send(JSON.stringify(msg));
            messageInput.value = '';
            messageInput.style.height = 'auto';
            clearError(messageErrorP);
            appendMessageToUI(msg, true);
            storeMessage(msg);
            updateChatListPreview(currentChatId, msg.text);
        } catch (error) {
            displayError(messageErrorP, 'Something went wrong, please try again.');
        }
    }

    function handleIncomingMessage(msg) {
        storeMessage(msg);
        if (msg.chat_id === currentChatId) {
            appendMessageToUI(msg, msg.from === currentUsername);
        }
        updateChatListPreview(msg.chat_id, msg.text);
    }

    // ── Chats ──────────────────────────────────────────────────────────────
    async function fetchChats() {
        if (!currentUsername) return;
        clearError(chatsErrorP);
        try {
            const fetched = await makeApiRequest(`/chats/${currentUsername}`);
            chats = {};
            if (Array.isArray(fetched)) {
                const storedPreviews = loadStoredPreviews();
                fetched.forEach(chat => {
                    let otherUser = 'Group Chat';
                    if (chat.members.length === 2)
                        otherUser = chat.members.find(m => m !== currentUsername) || chat.owner;
                    else if (chat.members.length === 1)
                        otherUser = currentUsername;
                    chats[chat.id] = { ...chat, otherUser, lastMessage: storedPreviews[chat.id] || null };
                });
            }
            renderChatList();
        } catch (error) {
            displayError(chatsErrorP, 'Something went wrong, please try again.');
            if (error.message.includes('401') || error.message.includes('Unauthorized')) handleLogout();
        }
    }

    function renderChatList() {
        chatListUl.innerHTML = '';
        if (Object.keys(chats).length === 0) {
            chatListUl.innerHTML =
                '<li class="text-on-surface-variant text-sm px-6 py-4">No chats yet. Search for a user to start one.</li>';
            return;
        }

        Object.values(chats).forEach(chat => {
            const li = document.createElement('li');
            li.dataset.chatId = chat.id;

            let displayName = chat.otherUser || chat.owner || `Chat ${chat.id.substring(0, 6)}`;
            if (displayName === currentUsername) displayName = 'Chat with Self';

            const initial   = displayName.charAt(0).toUpperCase();
            const isActive  = chat.id === currentChatId;
            const preview    = chat.lastMessage
                ? (chat.lastMessage.length > 45 ? chat.lastMessage.substring(0, 42) + '…' : chat.lastMessage)
                : '';
            const previewCls = 'font-body text-sm text-on-surface-variant truncate';

            li.className = [
                'flex items-center gap-6 p-6 rounded-lg cursor-pointer transition-colors group',
                isActive ? 'bg-surface-container-high' : 'hover:bg-surface-container'
            ].join(' ');

            li.innerHTML = `
                <div class="w-14 h-14 rounded-full bg-primary text-on-primary flex items-center justify-center shrink-0">
                    <span class="font-headline font-bold text-xl">${initial}</span>
                </div>
                <div class="flex-1 min-w-0 flex flex-col justify-center">
                    <h3 class="font-headline font-bold text-lg text-primary truncate mb-1">${displayName}</h3>
                    <p class="${previewCls}">${preview}</p>
                </div>
            `;

            li.addEventListener('click', () => selectChat(chat.id));
            chatListUl.appendChild(li);
        });
    }

    async function selectChat(chatId) {
        if (currentChatId === chatId) { showChatView(); return; }

        currentChatId = chatId;
        resetMessagesContainer();
        clearError(messageErrorP);

        document.querySelectorAll('#chat-list li').forEach(li => {
            const active = li.dataset.chatId === chatId;
            li.classList.toggle('bg-surface-container-high', active);
            li.classList.toggle('hover:bg-surface-container', !active);
        });

        const chat = chats[chatId];
        let displayName = chat
            ? (chat.otherUser || chat.owner || `Chat ${chatId.substring(0, 6)}`)
            : `Chat ${chatId.substring(0, 6)}`;
        if (displayName === currentUsername) displayName = 'Chat with Self';
        currentChatNameH2.textContent = displayName;

        showChatView();
        await fetchAndRenderMessages(chatId);
    }

    function storeMessage(msg) {
        const { chat_id } = msg;
        if (!messages[chat_id]) messages[chat_id] = [];
        if (!messages[chat_id].some(m => m.from === msg.from && m.text === msg.text)) {
            messages[chat_id].push(msg);
        }
    }

    async function fetchAndRenderMessages(chatId) {
        messagesContainer.innerHTML = '<p class="m-auto text-on-surface-variant text-sm">Loading…</p>';
        try {
            const fetched = await makeApiRequest(`/messages/${chatId}`);
            messagesContainer.innerHTML = '';
            messages[chatId] = [];

            if (Array.isArray(fetched) && fetched.length > 0) {
                fetched.forEach(msg => {
                    storeMessage({ from: msg.from, text: msg.text, chat_id: msg.chat_id });
                    appendMessageToUI({ from: msg.from, text: msg.text, chat_id: msg.chat_id }, msg.from === currentUsername);
                });
                // Seed the preview with the real last message from the server
                const last = fetched[fetched.length - 1];
                updateChatListPreview(chatId, last.text);
            } else if (!Array.isArray(fetched)) {
                messagesContainer.innerHTML = '<p class="m-auto text-on-surface-variant text-sm">Could not load messages.</p>';
            } else {
                messagesContainer.innerHTML = '<p class="m-auto text-on-surface-variant text-sm">No messages yet. Say hello!</p>';
            }
            scrollToBottom(messagesContainer);
        } catch (error) {
            displayError(messageErrorP, 'Something went wrong, please try again.');
            messagesContainer.innerHTML = '<p class="m-auto text-error text-sm">Something went wrong, please try again.</p>';
            if (error.message.includes('401') || error.message.includes('Unauthorized')) handleLogout();
        }
    }

    function appendMessageToUI(msg, isSent) {
        const placeholder = messagesContainer.querySelector('p');
        if (placeholder) placeholder.remove();

        const wrapper = document.createElement('div');
        wrapper.className = [
            'flex flex-col gap-1 max-w-2xl w-fit',
            isSent ? 'items-end self-end ml-auto' : 'items-start self-start'
        ].join(' ');

        const bubble = document.createElement('div');
        bubble.className = isSent
            ? 'bg-primary text-on-primary p-4 rounded-xl rounded-br-none ambient-shadow text-base leading-relaxed font-body'
            : 'bg-surface-container-highest text-on-surface p-4 rounded-xl rounded-bl-none shadow-sm text-base leading-relaxed font-body';
        bubble.textContent = msg.text;

        wrapper.appendChild(bubble);
        messagesContainer.appendChild(wrapper);
        scrollToBottom(messagesContainer);
    }

    function updateChatListPreview(chatId, text) {
        if (chats[chatId]) chats[chatId].lastMessage = text;
        persistPreview(chatId, text);
        const li = chatListUl.querySelector(`li[data-chat-id="${chatId}"]`);
        if (li) {
            const p = li.querySelector('p');
            if (p) {
                p.textContent = text.length > 45 ? text.substring(0, 42) + '…' : text;
                p.className = 'font-body text-sm text-on-surface-variant truncate';
            }
            chatListUl.prepend(li);
        }
    }

    function scrollToBottom(el) { el.scrollTop = el.scrollHeight; }

    // ── Search / new chat ──────────────────────────────────────────────────
    async function handleSearchUser() {
        const username = searchInput.value.trim();
        clearError(searchErrorP);
        clearError(searchInfoP);

        if (!username) { displayError(searchErrorP, 'Please enter a username.'); return; }
        if (username === currentUsername) {
            displayError(searchErrorP, 'You cannot start a chat with yourself.');
            return;
        }

        try {
            await makeApiRequest(`/user/${username}`, 'GET', null, false);
            displayInfo(searchInfoP, `User '${username}' found. Checking for existing chat…`);

            let existingId = null;
            for (const id in chats) {
                const c = chats[id];
                if (c.members.length === 2 && c.members.includes(currentUsername) && c.members.includes(username)) {
                    existingId = id; break;
                }
            }

            if (existingId) {
                displayInfo(searchInfoP, `Opening existing chat with '${username}'…`);
                searchInput.value = '';
                selectChat(existingId);
            } else {
                displayInfo(searchInfoP, `Starting new chat with '${username}'…`);
                await createNewChat(username);
                searchInput.value = '';
            }
        } catch (error) {
            if (error.message.includes('404') || error.message.includes('500'))
                displayError(searchErrorP, `User '${username}' not found.`);
            else
                displayError(searchErrorP, 'Something went wrong, please try again.');
        }
    }

    async function createNewChat(otherUsername) {
        try {
            const id = await makeApiRequest('/newChat', 'POST', {
                owner: currentUsername,
                members: [currentUsername, otherUsername]
            });
            if (id && typeof id === 'string') {
                chats[id] = { id, owner: currentUsername, members: [currentUsername, otherUsername], otherUser: otherUsername };
                messages[id] = [];
                renderChatList();
                selectChat(id);
            } else {
                throw new Error('Server did not return a valid chat ID.');
            }
        } catch (error) {
            displayError(searchErrorP, 'Something went wrong, please try again.');
        }
    }

    // ── Init ───────────────────────────────────────────────────────────────
    function initializeChatApp() {
        if (!authToken || !currentUsername) { showView('login'); return; }

        showView('app');
        loggedInUsernameSpan.textContent = currentUsername;
        resetMessagesContainer();
        currentChatNameH2.textContent = '';

        fetchChats();
        connectWebSocket();
    }

    // ── Focus search and go to chats list ─────────────────────────────────
    function focusSearch() {
        showChatsList();
        searchInput.focus();
    }

    // ── Event listeners ────────────────────────────────────────────────────
    loginForm.addEventListener('submit', handleLogin);
    registerForm.addEventListener('submit', handleRegister);

    if (logoutButton)   logoutButton.addEventListener('click', handleLogout);
    if (mobileLogoutBtn) mobileLogoutBtn.addEventListener('click', handleLogout);

    if (showRegisterBtn) showRegisterBtn.addEventListener('click', () => {
        clearError(loginErrorP); clearError(registerErrorP); clearError(registerSuccessP);
        registerForm.reset();
        showView('register');
    });
    if (showLoginBtn) showLoginBtn.addEventListener('click', () => {
        clearError(loginErrorP); clearError(registerErrorP); clearError(registerSuccessP);
        loginForm.reset();
        showView('login');
    });

    if (newChatBtn)       newChatBtn.addEventListener('click', focusSearch);
    if (mobileNewChatBtn) mobileNewChatBtn.addEventListener('click', focusSearch);
    if (navMessagesBtn)   navMessagesBtn.addEventListener('click', showChatsList);
    if (backToChatsBtn)   backToChatsBtn.addEventListener('click', showChatsList);

    // Search on Enter
    searchInput.addEventListener('keydown', (e) => {
        if (e.key === 'Enter') { e.preventDefault(); handleSearchUser(); }
    });

    // Send message
    messageForm.addEventListener('submit', (e) => {
        e.preventDefault();
        sendMessage(messageInput.value);
    });

    // Auto-grow textarea
    messageInput.addEventListener('input', () => {
        messageInput.style.height = 'auto';
        messageInput.style.height = Math.min(messageInput.scrollHeight, 128) + 'px';
    });

    // Send on Enter (Shift+Enter for newline)
    messageInput.addEventListener('keydown', (e) => {
        if (e.key === 'Enter' && !e.shiftKey) {
            e.preventDefault();
            sendMessage(messageInput.value);
        }
    });

    // ── Bootstrap ──────────────────────────────────────────────────────────
    initializeChatApp();
});
