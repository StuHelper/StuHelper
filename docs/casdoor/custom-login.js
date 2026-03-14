/*
 * StuHelper Casdoor SSO Login Page - Custom JavaScript
 *
 * Enhances the login page with StuHelper branding and smooth transitions.
 * Inject this JS in Casdoor Admin > Applications > StuHelper > Custom JS
 */

(function () {
  'use strict';

  // Wait for DOM ready
  function onReady(fn) {
    if (document.readyState !== 'loading') {
      fn();
    } else {
      document.addEventListener('DOMContentLoaded', fn);
    }
  }

  onReady(function () {
    // 1. Replace default title with StuHelper branding
    injectBranding();

    // 2. Add smooth entrance animation
    addEntranceAnimation();

    // 3. Enhance input focus labels
    enhanceInputLabels();
  });

  /**
   * Inject StuHelper branding into the login page header.
   * Targets common Casdoor selectors for the logo/title area.
   */
  function injectBranding() {
    var selectors = [
      '.login-form .logo',
      '.panel-module .logo',
      '.login-logo',
      '.login-form h1',
    ];

    for (var i = 0; i < selectors.length; i++) {
      var el = document.querySelector(selectors[i]);
      if (el && el.tagName === 'H1') {
        el.textContent = 'StuHelper';
        break;
      }
    }

    // Add subtitle if not present
    var titleEl = document.querySelector('.login-form h1, .panel-module h1, .login-form .title');
    if (titleEl && !titleEl.nextElementSibling?.classList?.contains('subtitle')) {
      var subtitle = document.createElement('p');
      subtitle.className = 'subtitle';
      subtitle.textContent = 'Sign in to your account';
      subtitle.style.cssText = 'color: rgba(255,255,255,0.5); font-size: 14px; text-align: center; margin: 4px 0 24px;';
      titleEl.insertAdjacentElement('afterend', subtitle);
    }
  }

  /**
   * Add entrance animation to the login card.
   */
  function addEntranceAnimation() {
    var card = document.querySelector('.login-form, .ant-card, .panel-module, .login-panel');
    if (!card) return;

    card.style.opacity = '0';
    card.style.transform = 'translateY(20px)';
    card.style.transition = 'opacity 0.5s ease, transform 0.5s ease';

    // Use requestAnimationFrame for smooth animation start
    requestAnimationFrame(function () {
      requestAnimationFrame(function () {
        card.style.opacity = '1';
        card.style.transform = 'translateY(0)';
      });
    });
  }

  /**
   * Enhance input labels with floating effect.
   */
  function enhanceInputLabels() {
    var inputs = document.querySelectorAll('.login-form input, .panel-module input');
    inputs.forEach(function (input) {
      input.addEventListener('focus', function () {
        this.parentElement?.classList?.add('input-focused');
      });
      input.addEventListener('blur', function () {
        if (!this.value) {
          this.parentElement?.classList?.remove('input-focused');
        }
      });
    });
  }
})();
