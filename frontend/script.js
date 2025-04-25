document.addEventListener('DOMContentLoaded', () => {
    // --- State Variables ---
    let authToken = localStorage.getItem('authToken');
    let currentUsername = localStorage.getItem('currentUsername');
    let webSocket = null;
    let currentChatId = null;
    let chats = {}; // Store chat details { id: { id, owner, members, otherUser? } }
    let messages = {}; // Store messages { chatId: [ { from, text, chatId } ] }

    // --- DOM Elements ---
    const authContainer = document.getElementById('auth-container');
    const appContainer = document.getElementById('app-container');
    const loginView = document.getElementById('login-view');
    const registerView = document.getElementById('register-view');

    // Forms
    const loginForm = document.getElementById('login-form');
    const registerForm = document.getElementById('register-form');
    const messageForm = document.getElementById('message-form');
    const searchButton = document.getElementById('search-button');
    const searchInput = document.getElementById('search-username');

    // Inputs
    const loginUsernameInput = document.getElementById('login-username');
    const loginPasswordInput = document.getElementById('login-password');
    const registerUsernameInput = document.getElementById('register-username');
    const registerPasswordInput = document.getElementById('register-password');
    const registerConfirmPasswordInput = document.getElementById('register-confirm-password');
    const messageInput = document.getElementById('message-input');

    // Display Areas
    const loggedInUsernameSpan = document.getElementById('logged-in-username');
    const chatListUl = document.getElementById('chat-list');
    const messagesContainer = document.getElementById('messages-container');
    const currentChatNameH2 = document.getElementById('current-chat-name');
    const messageInputContainer = document.getElementById('message-input-container');
    const noChatSelectedP = document.getElementById('no-chat-selected');

    // Buttons
    const logoutButton = document.getElementById('logout-button');
    const showRegisterLink = document.getElementById('show-register');
    const showLoginLink = document.getElementById('show-login');

    // Error/Info Messages
    const loginErrorP = document.getElementById('login-error');
    const registerErrorP = document.getElementById('register-error');
    const registerSuccessP = document.getElementById('register-success');
    const chatsErrorP = document.getElementById('chats-error');
    const messageErrorP = document.getElementById('message-error');
    const searchErrorP = document.getElementById('search-error');
    const searchInfoP = document.getElementById('search-info');

    // --- API Base URL ---
    // Assumes the backend is running on the same host/port
    // If different, replace with e.g., 'http://localhost:8080'
    const API_BASE_URL = window.location.origin;
    const WS_BASE_URL = API_BASE_URL.replace(/^http/, 'ws'); // ws:// or wss://

    // --- Helper Functions ---
    function displayError(element, message) {
        if (element) {
            element.textContent = message || 'An unexpected error occurred.';
            element.classList.remove('hidden');
        }
        console.error(message || 'An unexpected error occurred.');
    }

    function clearError(element) {
        if (element) {
            element.textContent = '';
            element.classList.add('hidden'); // Or just clear text if hidden class not used
        }
    }
    function displayInfo(element, message) {
        if (element) {
            element.textContent = message || '';
            element.classList.remove('hidden');
        }
    }

    function showView(viewToShow) {
        authContainer.classList.add('hidden');
        appContainer.classList.add('hidden');
        loginView.classList.add('hidden');
        registerView.classList.add('hidden');

        if (viewToShow === 'login') {
            authContainer.classList.remove('hidden');
            loginView.classList.remove('hidden');
        } else if (viewToShow === 'register') {
            authContainer.classList.remove('hidden');
            registerView.classList.remove('hidden');
        } else if (viewToShow === 'app') {
            appContainer.classList.remove('hidden');
        }
    }

    async function makeApiRequest(endpoint, method = 'GET', body = null, requiresAuth = true) {
        const url = `${API_BASE_URL}${endpoint}`;
        const headers = {
            'Content-Type': 'application/json',
        };
        if (requiresAuth && authToken) {
            headers['Authorization'] = `Bearer ${authToken}`;
        }

        const options = {
            method,
            headers,
        };
        if (body) {
            options.body = JSON.stringify(body);
        }

        try {
            const response = await fetch(url, options);

            if (!response.ok) {
                let errorMsg = `HTTP error! Status: ${response.status}`;
                try {
                    const errBody = await response.text(); // Try to get error text from backend
                    errorMsg = errBody || errorMsg;
                } catch (e) { /* Ignore if can't read body */ }
                throw new Error(errorMsg);
            }

            // Handle responses that might not have a body (e.g., 200 OK on register/delete)
            const contentType = response.headers.get("content-type");
            if (contentType && contentType.indexOf("application/json") !== -1) {
                return await response.json();
            } else {
                return await response.text(); // Or return null/true if no body expected
            }
        } catch (error) {
            console.error(`API Request Error (${method} ${endpoint}):`, error);
            throw error; // Re-throw to be caught by caller
        }
    }

    // --- Authentication ---
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
                console.log("Received token:", data); // Good for debugging
                authToken = data; // Assign the received string directly
                currentUsername = username; // Use the username from the input
                localStorage.setItem('authToken', authToken);
                localStorage.setItem('currentUsername', currentUsername);
                initializeChatApp(); // Proceed to the main app
            } else {
                // This branch will now correctly catch genuinely invalid responses
                console.error("Invalid response data received:", data);
                displayError(loginErrorP, 'Login failed: Invalid response format from server.');
            }

        } catch (error) {
            displayError(loginErrorP, `Login failed: ${error.message}`);
        }
    }

    async function handleRegister(event) {
        event.preventDefault();
        clearError(registerErrorP);
        clearError(registerSuccessP); // Clear success message too

        const username = registerUsernameInput.value.trim();
        const password = registerPasswordInput.value; // Don't trim passwords
        const confirmPassword = registerConfirmPasswordInput.value;

        if (!username || !password || !confirmPassword) {
            displayError(registerErrorP, 'All fields are required.');
            return;
        }
        if (password.length < 8) {
            displayError(registerErrorP, 'Password must be at least 8 characters long.');
            return;
        }
        if (password !== confirmPassword) {
            displayError(registerErrorP, 'Passwords do not match.');
            return;
        }

        try {
            // Backend returns 200 OK (empty body) on success
            await makeApiRequest('/register', 'POST', { username, password }, false);
            registerForm.reset();
            displayInfo(registerSuccessP, 'Registration successful! Please login.');
            // Optionally switch to login view automatically
            // showView('login');
        } catch (error) {
            displayError(registerErrorP, `Registration failed: ${error.message}`);
        }
    }

    function handleLogout() {
        authToken = null;
        currentUsername = null;
        localStorage.removeItem('authToken');
        localStorage.removeItem('currentUsername');

        if (webSocket) {
            webSocket.close();
            webSocket = null;
        }

        // Clear app state
        chats = {};
        messages = {};
        currentChatId = null;
        chatListUl.innerHTML = '';
        messagesContainer.innerHTML = '<p id="no-chat-selected">Select a chat from the list or start a new one by searching for a user.</p>';
        currentChatNameH2.textContent = 'Select a chat';
        messageInputContainer.classList.add('hidden');


        showView('login');
        // Clear form fields on logout
        loginForm.reset();
        registerForm.reset();
        clearError(loginErrorP);
        clearError(registerErrorP);
        clearError(registerSuccessP);


    }

    // --- WebSocket Communication ---
    function connectWebSocket() {
        if (webSocket && webSocket.readyState === WebSocket.OPEN) {
            console.log('WebSocket already connected.');
            return;
        }

        if (!authToken) {
            console.error('Cannot connect WebSocket without auth token.');
            handleLogout(); // Force logout if token is missing
            return;
        }

        // Construct WebSocket URL - needs Authorization, typically via query param or initial message
        // The Go backend's `authorize` middleware checks HTTP headers, *not* WebSocket protocols directly.
        // A common pattern is to pass the token in the URL, but this isn't ideal security-wise.
        // Let's *assume* the Go WS upgrade handler *implicitly* uses the context from the HTTP request,
        // meaning the `Authorization` header *during the upgrade request* is sufficient.
        const wsUrl = `${WS_BASE_URL}/ws`; // No token in URL needed based on Go code analysis

        console.log(`Attempting to connect WebSocket to ${wsUrl}`);
        // The `Upgrade` mechanism in Go likely uses the headers from the initial HTTP request
        // We rely on the browser sending the necessary cookies or headers implicitly if needed,
        // *but* standard WebSocket connection doesn't send arbitrary headers like Authorization.
        // *** CRITICAL POINT: *** Standard browser WebSocket API *cannot* set custom headers like `Authorization: Bearer ...`.
        // The Go backend *must* handle authorization differently for WS upgrade.
        // Common workarounds:
        // 1. Pass token as query parameter: `ws://.../ws?token=...` (Backend needs modification) - LESS SECURE
        // 2. Send token as the *first* message after connection (Backend needs modification)
        // 3. Rely on session cookies set during login (Backend needs modification)
        //
        // **Given the provided Go code:** It uses middleware (`s.authorize`) *before* `s.serveWS`.
        // This implies the HTTP upgrade request itself MUST be authorized. How does the frontend *make*
        // an authorized HTTP upgrade request? `new WebSocket()` doesn't allow headers.
        // This usually means the initial HTML page load (or an XHR prior to WS connection) sets a secure, HttpOnly cookie
        // that the browser *automatically* sends with the WS upgrade request.
        // Since the login endpoint only returns a token in the body, not setting cookies, the provided backend/frontend combo
        // *won't work* as is for authorizing the WebSocket connection standardly.
        //
        // **WORKAROUND/ASSUMPTION:** Let's *assume* for this example that either:
        //    a) The browser somehow magically sends the Authorization header (unlikely).
        //    b) The backend authorization check is flawed for `/ws` (possible).
        //    c) You'll modify the backend later to use cookies or a token-in-first-message strategy.
        // We proceed *as if* the connection will succeed based *only* on the HTTP middleware logic.

        webSocket = new WebSocket(wsUrl);


        webSocket.onopen = () => {
            console.log('WebSocket connection established.');
            // Maybe clear connection error messages here if any
        };

        webSocket.onmessage = (event) => {
            console.log('WebSocket message received:', event.data);
            try {
                const message = JSON.parse(event.data);
                // Validate message structure { from, chat_id, text }
                if (message.from && message.chat_id && message.text) {
                    handleIncomingMessage(message);
                } else {
                    console.warn('Received invalid message format:', message);
                }
            } catch (error) {
                console.error('Error parsing WebSocket message:', error);
            }
        };

        webSocket.onerror = (error) => {
            console.error('WebSocket error:', error);
            // Attempt to reconnect or inform user? For simplicity, just log.
            displayError(messageErrorP, 'WebSocket connection error. Real-time updates may fail.');

        };

        webSocket.onclose = (event) => {
            console.log('WebSocket connection closed:', event.code, event.reason);
            webSocket = null;
            // Attempt to reconnect after a delay?
            if (authToken) { // Only try reconnecting if user is supposed to be logged in
                displayError(messageErrorP, 'WebSocket disconnected. Attempting to reconnect...');
                setTimeout(connectWebSocket, 5000); // Reconnect after 5 seconds
            }
        };
    }

    function sendMessage(text) {
        if (!webSocket || webSocket.readyState !== WebSocket.OPEN) {
            displayError(messageErrorP, 'WebSocket is not connected. Cannot send message.');
            // Try to reconnect?
            connectWebSocket();
            return;
        }
        if (!currentChatId) {
            displayError(messageErrorP, 'No active chat selected.');
            return;
        }
        if (!text.trim()) {
            return; // Don't send empty messages
        }

        const message = {
            from: currentUsername,
            chat_id: currentChatId,
            text: text.trim(),
        };

        try {
            webSocket.send(JSON.stringify(message));
            // Clear input after sending
            messageInput.value = '';
            clearError(messageErrorP);

            // OPTIONAL: Add the sent message directly to the UI for immediate feedback
            // Or wait for the server to echo it back via onmessage
            // Let's add it immediately for better UX, assuming server confirms
            appendMessageToUI(message, true); // true indicates it's a sent message
            // Also store it locally immediately
            storeMessage(message);


        } catch (error) {
            displayError(messageErrorP, `Failed to send message: ${error.message}`);
            console.error('Error sending WebSocket message:', error);
        }
    }

    function handleIncomingMessage(message) {
        // Store the message
        storeMessage(message);

        // If the message is for the currently active chat, display it
        if (message.chat_id === currentChatId) {
            appendMessageToUI(message, message.from === currentUsername);
        }

        // Update chat list preview (optional, e.g., show last message)
        updateChatListPreview(message.chat_id, message.text);
    }

    // --- Chat & Message Management ---

    async function fetchChats() {
        if (!currentUsername) return;
        clearError(chatsErrorP);
        try {
            // API: GET /chats/{username}
            const fetchedChats = await makeApiRequest(`/chats/${currentUsername}`);
            chats = {}; // Reset local cache
            if (Array.isArray(fetchedChats)) {
                fetchedChats.forEach(chat => {
                    // Determine the 'other user' for display in 1-on-1 chats
                    let otherUser = 'Group Chat'; // Default for multi-person chats
                    if (chat.members.length === 2) {
                        otherUser = chat.members.find(member => member !== currentUsername) || chat.owner; // Find the other person
                    } else if (chat.members.length === 1) {
                        otherUser = currentUsername; // Chat with self
                    }
                    chats[chat.id] = { ...chat, otherUser };
                });
            }
            renderChatList();
        } catch (error) {
            displayError(chatsErrorP, `Failed to load chats: ${error.message}`);
            if (error.message.includes('401') || error.message.includes('Unauthorized')) {
                handleLogout(); // Token likely expired or invalid
            }
        }
    }

    function renderChatList() {
        chatListUl.innerHTML = ''; // Clear existing list
        if (Object.keys(chats).length === 0) {
            chatListUl.innerHTML = '<li>No chats found.</li>';
            return;
        }

        Object.values(chats).forEach(chat => {
            const li = document.createElement('li');
            li.dataset.chatId = chat.id;
            // Display logic: Use 'otherUser' if available, otherwise show members or owner
            let displayName = chat.otherUser || chat.owner || `Chat ${chat.id.substring(0, 6)}`;
            if (displayName === currentUsername) displayName = "Chat with Self"; // Special case

            li.textContent = displayName;
            li.title = `Members: ${chat.members.join(', ')} (Owner: ${chat.owner})`; // Tooltip
            if (chat.id === currentChatId) {
                li.classList.add('active');
            }
            // Store last message preview if needed (enhancement)
            if (chat.lastMessage) {
                const previewSpan = document.createElement('span');
                previewSpan.style.fontSize = '0.8em';
                previewSpan.style.color = 'var(--text-secondary-color)';
                previewSpan.style.display = 'block';
                previewSpan.textContent = chat.lastMessage.length > 25
                    ? chat.lastMessage.substring(0, 22) + '...'
                    : chat.lastMessage;
                li.appendChild(previewSpan);
            }

            li.addEventListener('click', () => selectChat(chat.id));
            chatListUl.appendChild(li);
        });
    }


    async function selectChat(chatId) {
        if (currentChatId === chatId) return; // Already selected

        currentChatId = chatId;
        messagesContainer.innerHTML = ''; // Clear previous messages
        messageInputContainer.classList.remove('hidden');
        noChatSelectedP.classList.add('hidden');
        clearError(messageErrorP); // Clear errors when switching chats

        // Highlight active chat in the list
        document.querySelectorAll('#chat-list li').forEach(li => {
            li.classList.toggle('active', li.dataset.chatId === chatId);
        });

        // Update chat header
        const chat = chats[chatId];
        if (chat) {
            let displayName = chat.otherUser || chat.owner || `Chat ${chatId.substring(0,6)}`;
            if (displayName === currentUsername) displayName = "Chat with Self";
            currentChatNameH2.textContent = displayName;
        } else {
            currentChatNameH2.textContent = `Chat ${chatId.substring(0, 6)}`;
        }


        // Fetch and render messages for this chat
        await fetchAndRenderMessages(chatId);
    }

    function storeMessage(message) {
        const { chat_id } = message;
        if (!messages[chat_id]) {
            messages[chat_id] = [];
        }
        // Avoid duplicates if message was added optimistically
        if (!messages[chat_id].some(m => m.from === message.from && m.text === message.text /* add timestamp check later */)) {
            messages[chat_id].push(message);
        }

        // Keep messages sorted (optional, if backend doesn't guarantee order)
        // messages[chat_id].sort((a, b) => a.timestamp - b.timestamp); // Requires timestamp from backend
    }


    async function fetchAndRenderMessages(chatId) {
        messagesContainer.innerHTML = 'Loading messages...'; // Loading indicator
        try {
            // API: GET /messages/{chatId}
            const fetchedMessages = await makeApiRequest(`/messages/${chatId}`);

            // Clear loading state and store/render
            messagesContainer.innerHTML = '';
            messages[chatId] = []; // Clear local cache for this chat before loading


            if (Array.isArray(fetchedMessages) && fetchedMessages.length > 0) {
                fetchedMessages.forEach(msg => {
                    // Assume backend message structure matches { from, chat_id, text }
                    // Add timestamp if backend provides it
                    storeMessage({ from: msg.from, text: msg.text, chat_id: msg.chat_id });
                    appendMessageToUI({ from: msg.from, text: msg.text, chat_id: msg.chat_id }, msg.from === currentUsername);
                });
            } else if (!Array.isArray(fetchedMessages)) {
                // Handle case where response is not an array (e.g., error string)
                console.warn(`Received non-array response for messages in chat ${chatId}:`, fetchedMessages);
                messagesContainer.innerHTML = '<p>No messages found or error loading.</p>';
            }
            else {
                messagesContainer.innerHTML = '<p>No messages in this chat yet.</p>';
            }
            scrollToBottom(messagesContainer);

        } catch (error) {
            displayError(messageErrorP, `Failed to load messages: ${error.message}`);
            messagesContainer.innerHTML = `<p class="error-message">Error loading messages: ${error.message}</p>`;
            if (error.message.includes('401') || error.message.includes('Unauthorized')) {
                handleLogout(); // Token likely expired or invalid
            } else if (error.message.includes('403')) { // Or Forbidden
                messagesContainer.innerHTML = `<p class="error-message">You are not authorized to view this chat.</p>`;
            }
        }
    }

    function appendMessageToUI(message, isSent) {
        // Ensure the 'no messages' placeholder is hidden
        const noMessages = messagesContainer.querySelector('p');
        if (noMessages && !noMessages.classList.contains('error-message')) {
            noMessages.remove();
        }

        const messageDiv = document.createElement('div');
        messageDiv.classList.add('message', isSent ? 'sent' : 'received');

        const senderSpan = document.createElement('span');
        senderSpan.classList.add('sender');
        senderSpan.textContent = isSent ? 'You' : message.from; // Show 'You' for sent, username for received

        const textDiv = document.createElement('div');
        textDiv.textContent = message.text;

        if (!isSent) { // Only show sender name for received messages if needed
            messageDiv.appendChild(senderSpan);
        }
        messageDiv.appendChild(textDiv);

        messagesContainer.appendChild(messageDiv);
        scrollToBottom(messagesContainer);
    }

    function updateChatListPreview(chatId, text) {
        const chatItem = chatListUl.querySelector(`li[data-chat-id="${chatId}"]`);
        if (chatItem) {
            let previewSpan = chatItem.querySelector('span');
            if (!previewSpan) {
                previewSpan = document.createElement('span');
                previewSpan.style.fontSize = '0.8em';
                previewSpan.style.color = 'var(--text-secondary-color)';
                previewSpan.style.display = 'block';
                chatItem.appendChild(previewSpan);
            }
            previewSpan.textContent = text.length > 25 ? text.substring(0, 22) + '...' : text;

            // Optionally move the updated chat to the top of the list
            chatListUl.prepend(chatItem);
        }
        // Store last message in chat data for re-rendering
        if (chats[chatId]) {
            chats[chatId].lastMessage = text;
        }
    }


    function scrollToBottom(element) {
        element.scrollTop = element.scrollHeight;
    }

    // --- User Search and New Chat ---
    async function handleSearchUser() {
        const searchUsername = searchInput.value.trim();
        clearError(searchErrorP);
        clearError(searchInfoP); // Clear previous info message

        if (!searchUsername) {
            displayError(searchErrorP, "Please enter a username to search.");
            return;
        }
        if (searchUsername === currentUsername) {
            displayError(searchErrorP, "You cannot start a chat with yourself using search. Use the 'Chat with Self' option if needed or find your existing self-chat.");
            return;
        }


        try {
            // 1. Check if user exists (API: GET /user/{username})
            // This endpoint doesn't require auth in the Go code provided
            await makeApiRequest(`/user/${searchUsername}`, 'GET', null, false);
            displayInfo(searchInfoP, `User '${searchUsername}' found. Checking for existing chat...`);


            // 2. Check if a 1-on-1 chat already exists
            let existingChatId = null;
            for (const chatId in chats) {
                const chat = chats[chatId];
                if (chat.members.length === 2 &&
                    chat.members.includes(currentUsername) &&
                    chat.members.includes(searchUsername)) {
                    existingChatId = chatId;
                    break;
                }
            }

            if (existingChatId) {
                // 3a. Chat exists, select it
                displayInfo(searchInfoP, `Chat with '${searchUsername}' already exists. Opening...`);
                selectChat(existingChatId);
                searchInput.value = ''; // Clear search input
            } else {
                // 3b. Chat doesn't exist, create it
                displayInfo(searchInfoP, `Starting new chat with '${searchUsername}'...`);
                await createNewChat(searchUsername);
                searchInput.value = ''; // Clear search input

            }

        } catch (error) {
            if (error.message.includes('404') || error.message.includes('500')) { // Assuming 500 might mean "user not found" from GetUser
                displayError(searchErrorP, `User '${searchUsername}' not found.`);
            } else {
                displayError(searchErrorP, `Error searching for user: ${error.message}`);
            }
        }
    }

    async function createNewChat(otherUsername) {
        try {
            // API: POST /newChat
            // Body: { owner: "currentUser", members: ["currentUser", "otherUser"] }
            const newChatData = await makeApiRequest('/newChat', 'POST', {
                owner: currentUsername,
                members: [currentUsername, otherUsername]
            });

            // Check if 'newChatData' is a non-empty string (which is the chat ID itself)
            if (newChatData && typeof newChatData === 'string') {
                const newChatId = newChatData; // Assign the received string directly
                console.log("Received new chat ID:", newChatId); // Good for debugging
                displayInfo(searchInfoP, `Chat created successfully (ID: ${newChatId.substring(0,8)}...).`); // Show partial ID maybe

                // Add the new chat to our local state *before* fetching all chats again
                chats[newChatId] = {
                    id: newChatId,
                    owner: currentUsername,
                    members: [currentUsername, otherUsername],
                    otherUser: otherUsername // Set this immediately
                };
                messages[newChatId] = []; // Initialize messages array

                await renderChatList(); // Re-render list with new chat
                selectChat(newChatId); // Automatically select the new chat
            } else {
                // This branch will now correctly catch genuinely invalid responses
                console.error("Invalid response data received for new chat:", newChatData);
                throw new Error("Server did not return a valid chat ID format.");
            }
        } catch (error) {
            displayError(searchErrorP, `Failed to create chat: ${error.message}`);
        }
    }


    // --- Initialization ---
    function initializeChatApp() {
        if (!authToken || !currentUsername) {
            showView('login');
            return;
        }

        showView('app');
        loggedInUsernameSpan.textContent = currentUsername;
        messageInputContainer.classList.add('hidden'); // Hide input until chat selected
        currentChatNameH2.textContent = 'Select a chat';
        messagesContainer.innerHTML = '<p id="no-chat-selected">Select a chat from the list or start a new one by searching for a user.</p>';


        // Fetch initial data
        fetchChats();

        // Connect WebSocket
        connectWebSocket();
    }

    // --- Event Listeners ---
    loginForm.addEventListener('submit', handleLogin);
    registerForm.addEventListener('submit', handleRegister);
    logoutButton.addEventListener('click', handleLogout);
    messageForm.addEventListener('submit', (e) => {
        e.preventDefault();
        sendMessage(messageInput.value);
    });
    searchButton.addEventListener('click', handleSearchUser);
    // Optional: Allow search on Enter key press in search input
    searchInput.addEventListener('keypress', (e) => {
        if (e.key === 'Enter') {
            handleSearchUser();
        }
    });


    showRegisterLink.addEventListener('click', (e) => {
        e.preventDefault();
        clearError(loginErrorP);
        clearError(registerErrorP);
        clearError(registerSuccessP);
        registerForm.reset();
        showView('register');
    });

    showLoginLink.addEventListener('click', (e) => {
        e.preventDefault();
        clearError(loginErrorP);
        clearError(registerErrorP);
        clearError(registerSuccessP);
        loginForm.reset();
        showView('login');
    });

    // --- Initial Check ---
    initializeChatApp(); // Check for stored token on load
});