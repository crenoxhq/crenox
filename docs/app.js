/* ==========================================================================
   Crenox Landing Page Logic
   Features: Minimal scroll fade-in reveals, Theme toggle, Install hub tabs, Quick copy.
   ========================================================================== */

// ─── Immediate Theme Restoration (Prevents Theme Flash) ────────────────────
(function() {
    try {
        const savedTheme = localStorage.getItem('crenox_theme');
        const prefersDark = window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches;
        const initialTheme = savedTheme || (prefersDark ? 'dark' : 'light');
        document.documentElement.setAttribute('data-theme', initialTheme);
        if (document.body) {
            document.body.setAttribute('data-theme', initialTheme);
        }
    } catch (e) {}
})();

document.addEventListener('DOMContentLoaded', () => {

    // ─── Theme Toggle Handlers ───────────────────────────────────────────
    const themeToggleBtns = document.querySelectorAll('.theme-toggle-btn, #themeToggle');
    themeToggleBtns.forEach(btn => {
        btn.addEventListener('click', (e) => {
            e.preventDefault();
            e.stopPropagation();
            const current = document.documentElement.getAttribute('data-theme') || 'light';
            const next = current === 'dark' ? 'light' : 'dark';
            document.documentElement.setAttribute('data-theme', next);
            if (document.body) {
                document.body.setAttribute('data-theme', next);
            }
            try {
                localStorage.setItem('crenox_theme', next);
            } catch (err) {}
        });
    });

    // ─── Install Hub Tab Switching ─────────────────────────────────────────
    const installHubs = document.querySelectorAll('.install-hub');
    installHubs.forEach(hub => {
        const tabs = hub.querySelectorAll('.install-tab-btn');
        const panels = hub.querySelectorAll('.install-tab-panel');

        tabs.forEach(tab => {
            tab.addEventListener('click', (e) => {
                e.preventDefault();
                const targetId = tab.getAttribute('data-tab');
                tabs.forEach(t => t.classList.remove('active'));
                panels.forEach(p => p.classList.remove('active'));

                tab.classList.add('active');
                const targetPanel = hub.querySelector('#' + targetId);
                if (targetPanel) {
                    targetPanel.classList.add('active');
                }
            });
        });
    });

    // ─── Quick Install Copy to Clipboard Handler ──────────────────────────
    const copyBtns = document.querySelectorAll('.copy-btn');
    copyBtns.forEach(btn => {
        btn.addEventListener('click', (e) => {
            e.preventDefault();
            e.stopPropagation();
            const copyTarget = btn.getAttribute('data-copy') || 
                               (btn.closest('.install-cmd-row') ? btn.closest('.install-cmd-row').querySelector('code').textContent.trim() : '');
            if (!copyTarget) return;

            const originalHTML = btn.innerHTML;
            const successText = 'Copied!';

            const notifySuccess = () => {
                btn.textContent = successText;
                btn.classList.add('copied');
                setTimeout(() => {
                    btn.innerHTML = originalHTML;
                    btn.classList.remove('copied');
                }, 2000);
            };

            if (navigator.clipboard && navigator.clipboard.writeText) {
                navigator.clipboard.writeText(copyTarget).then(notifySuccess).catch(() => {
                    fallbackCopy(copyTarget, notifySuccess);
                });
            } else {
                fallbackCopy(copyTarget, notifySuccess);
            }
        });
    });

    function fallbackCopy(text, callback) {
        const textarea = document.createElement('textarea');
        textarea.value = text;
        textarea.style.position = 'fixed';
        textarea.style.opacity = '0';
        document.body.appendChild(textarea);
        textarea.select();
        try {
            document.execCommand('copy');
            if (callback) callback();
        } catch (e) {}
        document.body.removeChild(textarea);
    }

    // ─── Scroll Entry Animations (IntersectionObserver) ───────────────────
    const fadeElements = document.querySelectorAll('.scroll-fade');

    const observerOptions = {
        root: null,
        threshold: 0.05,
        rootMargin: '0px 0px -40px 0px'
    };

    const observer = new IntersectionObserver((entries, observer) => {
        entries.forEach(entry => {
            if (entry.isIntersecting) {
                entry.target.classList.add('active');
                observer.unobserve(entry.target);
            }
        });
    }, observerOptions);

    fadeElements.forEach(el => {
        observer.observe(el);
    });

    // ─── Hamburger Menu Toggle ───────────────────────────────────────────
    const menuToggle = document.getElementById('menuToggle');
    const navBar = document.querySelector('.nav-bar');

    if (menuToggle && navBar) {
        menuToggle.addEventListener('click', () => {
            menuToggle.classList.toggle('active');
            navBar.classList.toggle('active');
        });

        // Close menu when clicking any nav link
        const navLinks = document.querySelectorAll('.nav-link');
        navLinks.forEach(link => {
            link.addEventListener('click', () => {
                menuToggle.classList.remove('active');
                navBar.classList.remove('active');
            });
        });

        // Close menu when clicking outside
        document.addEventListener('click', (e) => {
            if (navBar.classList.contains('active') && !navBar.contains(e.target)) {
                menuToggle.classList.remove('active');
                navBar.classList.remove('active');
            }
        });
    }

    // ─── Dynamic GitHub Release Version Fetcher ──────────────────────────
    fetch('https://api.github.com/repos/crenoxhq/crenox/releases/latest')
        .then(response => {
            if (response.ok) return response.json();
            throw new Error('Network response not ok');
        })
        .then(data => {
            const version = data.tag_name; // e.g. "v2.1.0"
            document.querySelectorAll('.latest-version').forEach(el => {
                el.textContent = version;
            });
        })
        .catch(err => {
            console.warn('Crenox: Could not dynamically resolve latest version from GitHub API, using fallback.', err);
        });

});
