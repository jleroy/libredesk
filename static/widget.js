/**
 * Libredesk Chat Widget
 * Embeddable chat widget for websites
 */
(function () {
    'use strict';

    if (window.__libredeskWidgetLoaded) {
        return;
    }
    window.__libredeskWidgetLoaded = true;

    class Libredesk {
        constructor(config = {}) {
            if (!config.baseURL) {
                throw new Error('baseURL is required');
            }
            if (!config.inboxID) {
                throw new Error('inboxID is required');
            }

            this.IFRAME_BORDER_RADIUS = '24px';
            this.IFRAME_BOX_SHADOW = '0 4px 24px rgba(0,0,0,0.12)';
            this.IFRAME_WIDTH = '400px';
            this.IFRAME_HEIGHT = '700px';
            this.EXPANDED_WIDTH = '750px';
            this.MOBILE_BREAKPOINT = 600;

            this.config = config;
            this.iframe = null;
            this.toggleButton = null;
            this.widgetButtonWrapper = null;
            this.unreadBadge = null;
            this.isChatVisible = false;
            this.widgetSettings = null;
            this.unreadCount = 0;
            this.isMobile = window.innerWidth <= this.MOBILE_BREAKPOINT;
            this.isExpanded = false;
            this.hideLauncher = config.hideLauncher || false;
            this._onShowCallback = null;
            this._onHideCallback = null;
            this._onUnreadCountChangeCallback = null;
            this._boundHandleMessage = (e) => this.handleMessage(e);
            this._boundHandleResize = () => this.handleResize();
            this.init();
        }

        postToIframe (data) {
            if (this.iframe && this.iframe.contentWindow) {
                this.iframe.contentWindow.postMessage(data, '*');
            }
        }

        formatBadgeCount (count) {
            return count > 99 ? '99+' : count.toString();
        }

        async init () {
            try {
                await this.fetchWidgetSettings();
                this.createElements();
                this.setLauncherPosition();
                this.widgetButtonWrapper.style.display = 'none';
                this.iframe.addEventListener('load', () => {
                    this.sendMobileState();
                });
                this.setupMobileDetection();
                this.setupEventListeners();
                this.startPageTracking();
            } catch (error) {
                console.error('Failed to initialize Libredesk Widget:', error);
            }
        }

        async fetchWidgetSettings () {
            try {
                const response = await fetch(`${this.config.baseURL}/api/v1/widget/chat/settings/launcher?inbox_id=${this.config.inboxID}`);

                if (!response.ok) {
                    throw new Error(`Error fetching widget settings. Status: ${response.status}`);
                }

                const result = await response.json();

                if (result.status !== 'success') {
                    throw new Error('Failed to fetch widget settings');
                }

                this.widgetSettings = result.data;
            } catch (error) {
                console.error('Error fetching widget settings:', error);
                throw error;
            }
        }

        createElements () {
            const launcher = this.widgetSettings.launcher;
            const colors = this.widgetSettings.colors;

            this.toggleButton = document.createElement('div');
            this.toggleButton.style.cssText = `
                position: fixed;
                cursor: pointer;
                z-index: 9999;
                width: 60px;
                height: 60px;
                background-color: ${colors.primary};
                border-radius: 50%;
                display: flex;
                justify-content: center;
                align-items: center;
                box-shadow: 0 2px 12px rgba(0,0,0,0.15);
                transition: transform 0.3s ease;
            `;

            this.iconContainer = document.createElement('div');
            this.iconContainer.style.cssText = `
                width: 100%;
                height: 100%;
                display: flex;
                justify-content: center;
                align-items: center;
                transition: transform 0.3s ease;
            `;

            if (launcher.logo_url) {
                this.defaultIcon = document.createElement('img');
                this.defaultIcon.src = launcher.logo_url;
                this.defaultIcon.style.cssText = `
                    width: 100%;
                    height: 100%;
                    border-radius: 50%;
                    object-fit: cover;
                `;
                this.iconContainer.appendChild(this.defaultIcon);
            }

            this.arrowIcon = document.createElement('div');
            const svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
            svg.setAttribute('width', '24');
            svg.setAttribute('height', '24');
            svg.setAttribute('viewBox', '0 0 24 24');
            svg.setAttribute('fill', 'none');
            const path = document.createElementNS('http://www.w3.org/2000/svg', 'path');
            path.setAttribute('d', 'M7 10L12 15L17 10');
            path.setAttribute('stroke', 'white');
            path.setAttribute('stroke-width', '2');
            path.setAttribute('stroke-linecap', 'round');
            path.setAttribute('stroke-linejoin', 'round');
            svg.appendChild(path);
            this.arrowIcon.appendChild(svg);
            this.arrowIcon.style.cssText = `
                width: 100%;
                height: 100%;
                display: none;
                justify-content: center;
                align-items: center;
            `;
            this.iconContainer.appendChild(this.arrowIcon);

            this.toggleButton.appendChild(this.iconContainer);

            this.unreadBadge = document.createElement('div');
            this.unreadBadge.style.cssText = `
                position: absolute;
                top: -5px;
                right: -5px;
                background-color: #ef4444;
                color: white;
                border-radius: 50%;
                width: 20px;
                height: 20px;
                display: none;
                justify-content: center;
                align-items: center;
                font-size: 12px;
                font-weight: bold;
                font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
                border: 2px solid white;
                box-sizing: border-box;
                z-index: 10000;
            `;

            const widgetButtonWrapper = document.createElement('div');
            widgetButtonWrapper.style.cssText = `
                position: fixed;
                z-index: 9999;
            `;

            widgetButtonWrapper.appendChild(this.toggleButton);
            widgetButtonWrapper.appendChild(this.unreadBadge);
            this.toggleButton.style.position = 'relative';
            this.widgetButtonWrapper = widgetButtonWrapper;

            this.iframe = document.createElement('iframe');
            this.iframe.src = `${this.config.baseURL}/widget?inbox_id=${this.config.inboxID}`;
            this.iframe.style.cssText = `
                position: fixed;
                border: none;
                border-radius: ${this.IFRAME_BORDER_RADIUS};
                box-shadow: ${this.IFRAME_BOX_SHADOW};
                z-index: 9999;
                width: ${this.IFRAME_WIDTH};
                height: ${this.IFRAME_HEIGHT};
                transition: all 0.3s ease;
                display: none;
            `;

            document.body.appendChild(this.widgetButtonWrapper);
            document.body.appendChild(this.iframe);
        }

        sendMobileState () {
            this.isMobile = window.innerWidth <= this.MOBILE_BREAKPOINT;
            this.postToIframe({
                type: 'SET_MOBILE_STATE',
                isMobile: this.isMobile
            });
        }

        sendPageInfo () {
            this.postToIframe({
                type: 'PAGE_VISIT',
                url: window.location.href,
                title: document.title || ''
            });
        }

        setLauncherPosition () {
            const spacing = this.widgetSettings.launcher.spacing;
            const side = this.widgetSettings.launcher.position === 'right' ? 'right' : 'left';

            this.widgetButtonWrapper.style.bottom = `${spacing.bottom}px`;
            this.widgetButtonWrapper.style[side] = `${spacing.side}px`;

            this.iframe.style.bottom = `${spacing.bottom + 80}px`;
            this.iframe.style[side] = `${spacing.side}px`;
        }

        handleMessage (event) {
            if (event.source !== this.iframe.contentWindow) return;

            switch (event.data.type) {
                case 'VUE_APP_READY':
                    this.handleVueAppReady();
                    break;
                case 'CLOSE_WIDGET':
                    this.hideChat();
                    break;
                case 'UPDATE_UNREAD_COUNT':
                    this.updateUnreadCount(event.data.count);
                    break;
                case 'EXPAND_WIDGET':
                    this.expandWidget();
                    break;
                case 'COLLAPSE_WIDGET':
                    this.collapseWidget();
                    break;
                case 'REQUEST_PAGE_INFO':
                    this.sendPageInfo();
                    break;
            }
        }

        setupEventListeners () {
            this.toggleButton.addEventListener('click', () => this.toggle());
            window.addEventListener('message', this._boundHandleMessage);
        }

        handleResize () {
            this.sendMobileState();
            if (this.isChatVisible) {
                this.showChat();
            }
        }

        setupMobileDetection () {
            window.addEventListener('resize', this._boundHandleResize);
            window.addEventListener('orientationchange', this._boundHandleResize);
        }

        handleVueAppReady () {
            this.sendMobileState();
            if (!this.hideLauncher) {
                this.widgetButtonWrapper.style.display = '';
            }

            if (this.config.userJWT) {
                this.postToIframe({
                    type: 'SET_JWT_TOKEN',
                    jwt: this.config.userJWT
                });
            }
        }

        toggle () {
            if (this.isChatVisible) {
                this.hideChat();
                this.postToIframe({ type: 'WIDGET_CLOSED' });
            } else {
                this.showChat();
                this.postToIframe({ type: 'WIDGET_OPENED' });
            }
        }

        showChat () {
            if (this.iframe) {
                this.isMobile = window.innerWidth <= this.MOBILE_BREAKPOINT;
                this.iframe.style.display = 'block';
                this.iframe.style.position = 'fixed';

                if (this.isMobile) {
                    this.iframe.style.top = '0';
                    this.iframe.style.left = '0';
                    this.iframe.style.right = '0';
                    this.iframe.style.bottom = '0';
                    this.iframe.style.width = '100vw';
                    this.iframe.style.height = '100dvh';
                    this.iframe.style.borderRadius = '0';
                    this.iframe.style.boxShadow = 'none';
                    this.widgetButtonWrapper.style.display = 'none';
                } else {
                    this.iframe.style.width = this.IFRAME_WIDTH;
                    this.iframe.style.borderRadius = this.IFRAME_BORDER_RADIUS;
                    this.iframe.style.boxShadow = this.IFRAME_BOX_SHADOW;
                    this.iframe.style.top = '';
                    this.iframe.style.left = '';
                    this.widgetButtonWrapper.style.display = '';

                    if (this.isExpanded) {
                        this.iframe.style.width = this.EXPANDED_WIDTH;
                        this.iframe.style.height = 'calc(100vh - 40px)';
                        this.iframe.style.bottom = '20px';
                    } else {
                        this.iframe.style.height = this.IFRAME_HEIGHT;
                        this.setLauncherPosition();
                    }
                }
                this.isChatVisible = true;
                this.toggleButton.style.transform = 'scale(0.9)';
                this.unreadBadge.style.display = 'none';

                if (this.defaultIcon) this.defaultIcon.style.display = 'none';
                this.arrowIcon.style.display = 'flex';

                if (this._onShowCallback) this._onShowCallback();
            }
        }

        hideChat () {
            if (this.iframe) {
                this.iframe.style.display = 'none';
                this.isChatVisible = false;
                this.toggleButton.style.transform = 'scale(1)';
                this.widgetButtonWrapper.style.display = '';

                if (this.defaultIcon) this.defaultIcon.style.display = 'block';
                this.arrowIcon.style.display = 'none';

                if (this.unreadCount > 0) {
                    this.unreadBadge.textContent = this.formatBadgeCount(this.unreadCount);
                    this.unreadBadge.style.display = 'flex';
                }

                if (this._onHideCallback) this._onHideCallback();
            }
        }

        updateUnreadCount (count) {
            this.unreadCount = count;
            if (this._onUnreadCountChangeCallback) this._onUnreadCountChangeCallback(count);

            if (count > 0 && !this.isChatVisible) {
                this.unreadBadge.textContent = this.formatBadgeCount(count);
                this.unreadBadge.style.display = 'flex';
            } else {
                this.unreadBadge.style.display = 'none';
            }
        }

        expandWidget () {
            if (this.iframe && this.isChatVisible && !this.isMobile) {
                this.isExpanded = true;
                this.iframe.style.width = this.EXPANDED_WIDTH;
                this.iframe.style.height = 'calc(100vh - 40px)';
                this.iframe.style.bottom = '20px';
                this.postToIframe({ type: 'WIDGET_EXPANDED', isExpanded: true });
            }
        }

        collapseWidget () {
            if (this.iframe && this.isChatVisible && !this.isMobile) {
                this.isExpanded = false;
                this.iframe.style.width = this.IFRAME_WIDTH;
                this.iframe.style.height = this.IFRAME_HEIGHT;
                this.iframe.style.top = '';
                this.setLauncherPosition();
                this.postToIframe({ type: 'WIDGET_EXPANDED', isExpanded: false });
            }
        }

        startPageTracking () {
            this._lastPageURL = '';
            this._origPushState = history.pushState;
            this._origReplaceState = history.replaceState;

            const self = this;
            const onPageChange = () => {
                const url = window.location.href;
                if (url === self._lastPageURL) return;
                self._lastPageURL = url;
                // Defer to let SPA frameworks update document.title after route change.
                setTimeout(() => { self.sendPageInfo(); }, 100);
            };

            history.pushState = function () {
                self._origPushState.apply(this, arguments);
                onPageChange();
            };
            history.replaceState = function () {
                self._origReplaceState.apply(this, arguments);
                onPageChange();
            };

            this._onPopState = onPageChange;
            this._onHashChange = onPageChange;
            window.addEventListener('popstate', this._onPopState);
            window.addEventListener('hashchange', this._onHashChange);

            this._pageTrackInterval = setInterval(onPageChange, 7000);
            onPageChange();
        }

        stopPageTracking () {
            if (this._origPushState) history.pushState = this._origPushState;
            if (this._origReplaceState) history.replaceState = this._origReplaceState;
            if (this._onPopState) window.removeEventListener('popstate', this._onPopState);
            if (this._onHashChange) window.removeEventListener('hashchange', this._onHashChange);
            if (this._pageTrackInterval) clearInterval(this._pageTrackInterval);
        }

        setUser (jwt) {
            this.postToIframe({ type: 'SET_JWT_TOKEN', jwt: jwt });
        }

        logout () {
            this.postToIframe({ type: 'CLEAR_SESSION' });
        }

        destroy () {
            this.stopPageTracking();
            window.removeEventListener('message', this._boundHandleMessage);
            window.removeEventListener('resize', this._boundHandleResize);
            window.removeEventListener('orientationchange', this._boundHandleResize);
            if (this.widgetButtonWrapper) {
                document.body.removeChild(this.widgetButtonWrapper);
                this.widgetButtonWrapper = null;
                this.toggleButton = null;
                this.unreadBadge = null;
            }
            if (this.iframe) {
                document.body.removeChild(this.iframe);
                this.iframe = null;
            }
            this.isChatVisible = false;
            this._onShowCallback = null;
            this._onHideCallback = null;
            this._onUnreadCountChangeCallback = null;
        }
    }

    Libredesk.prototype.show = Libredesk.prototype.showChat;
    Libredesk.prototype.hide = Libredesk.prototype.hideChat;
    Libredesk.prototype.isVisible = function () { return this.isChatVisible; };
    Libredesk.prototype.onShow = function (fn) { this._onShowCallback = fn; };
    Libredesk.prototype.onHide = function (fn) { this._onHideCallback = fn; };
    Libredesk.prototype.onUnreadCountChange = function (fn) { this._onUnreadCountChangeCallback = fn; fn(this.unreadCount); };

    window.Libredesk = Libredesk;

    window.initLibredesk = function (config = {}) {
        if (window.Libredesk && window.Libredesk instanceof Libredesk) {
            console.warn('Libredesk Widget is already initialized');
            return window.Libredesk;
        }
        window.Libredesk = new Libredesk(config);
        return window.Libredesk;
    };

    if (window.LibredeskSettings) {
        window.initLibredesk(window.LibredeskSettings);
    }

})();
