import { Component, ViewChild } from '@angular/core';
import { Router, ActivatedRoute, NavigationEnd } from '@angular/router';
import { AuthService } from '../guards/auth.service';
import { Http } from '@angular/http';
import { getHttpRequestOptions } from '../utils/httpUtil';
import { TableBaseComponent } from '../base-components/table.base.component';
import { handleAuthError } from '../utils/authUtil';
function getUrlPath(url) {
  const parser = document.createElement('a');
  parser.href = url;
  return parser.pathname;
}

const KEY_RETURN = 13;
const KEY_ENTER = 14;
const KEY_LEFT = 37;
const KEY_UP = 38;
const KEY_RIGHT = 39;
const KEY_DOWN = 40;

@Component({
  selector: 'app-header',
  templateUrl: './app.header.component.html',
  styleUrls: ['./app.header.component.css'],
})
export class AppHeaderComponent extends TableBaseComponent {
  @ViewChild('searchbox') searchbox;

  title = 'app';

  hamburgerExpanded = false;
  menuExpanded = false;
  isDashboard = false;
  changePasswordExpanded = false;
  currPsMatch = true;
  newPasswordEmpty = false;
  newPasswordMatch = true;
  newPs = '';
  changePasswordSuccess = false;
  changePasswordFail = false;
  btnDisabled = false;
  showChangePassword = localStorage['sherlock_refresh_token'] ? false : true;
  sub = null;

  menus = [
    {
      item: 'Applications',
      url: 'applications',
      expanded: false,
    },
    {
      item: 'Projects',
      url: 'projects',
      expanded: false,
    },

    {
      item: 'Data Streams',
      url: 'datastreams',
      expanded: false,
    },
    {
      item: 'Scripts',
      url: 'scripts',
      expanded: false,
    },
    {
      item: 'Edges',
      url: 'edges',
      expanded: false,
    },

    {
      item: 'Data Sources',
      url: 'datasources',
      expanded: false,
    },
    {
      item: 'Users',
      url: 'users',
      expanded: false,
    },
    {
      item: 'Categories',
      url: 'categories',
      expanded: false,
    },
    {
      item: 'Logs',
      url: 'log',
      expanded: false,
    },
    {
      item: 'Settings',
      url: 'settings',
      expanded: false,
    },
    {
      item: 'Cloud Profiles',
      url: 'clouds',
      expanded: false,
    },
    {
      item: 'Container Registry Profiles',
      url: 'container',
      expanded: false,
    },
  ];

  searchTerm = 'dashboard';
  routerEventSubscription = null;
  focused = false;
  enableNavMenuClickOutsideDetection = false;

  knownSuggestions = [];
  suggestions = [];
  selectedSuggestionIndex = 0;

  username = '';

  constructor(
    private route: ActivatedRoute,
    router: Router,
    private authService: AuthService,
    private http: Http
  ) {
    super(router);
    this.sub = this.router.events.subscribe(event => {
      if (event instanceof NavigationEnd) {
        this.getProjectInfo();
      }
    });
    this.knownSuggestions = this.menus.map(m => ({
      name: m.item,
      route: m.url,
    }));
    this.username = this.authService.getUser();
  }
  getProjectInfo() {
    let promise = [];
    promise.push(
      this.http.get('/v1/projects', getHttpRequestOptions()).toPromise()
    );
    promise.push(
      this.http.get('/v1/users', getHttpRequestOptions()).toPromise()
    );

    Promise.all(promise).then(
      response => {
        const projects = response[0].json();
        const users = response[1].json();
        this._totalProjects = 0;
        projects.forEach(p => {
          if (p.users) {
            p.users.forEach(pUser => {
              users.some(u => {
                if (u.id === pUser.userId) {
                  if (
                    u.email.trim().toLowerCase() ===
                    this._sherlockUsername.trim().toLowerCase()
                  ) {
                    if (u.role === 'USER') localStorage['sherlock_role'] = '';
                    else localStorage['sherlock_role'] = 'infra_admin';
                    this._sherlockRole = localStorage['sherlock_role'];
                    this._totalProjects++;
                  }
                }
              });
            });
          }
        });
      },
      error => {
        handleAuthError(null, error, this.router, this.http, () =>
          this.getProjectInfo()
        );
      }
    );
  }

  onClickHamburger() {
    this.hamburgerExpanded = !this.hamburgerExpanded;
    if (this.hamburgerExpanded) {
      setTimeout(() => {
        this.enableNavMenuClickOutsideDetection = true;
      });
    } else {
      this.enableNavMenuClickOutsideDetection = false;
    }
  }

  onMouseOverMenuItem(menu) {
    this.hideAllSubMenus();
    if (menu && (menu.submenu || menu.submenuGroup)) {
      menu.expanded = true;
    }
  }

  hideAllSubMenus() {
    this.menus.forEach(m => (m.expanded = false));
  }

  onClickSubMenu(sm) {
    this.hamburgerExpanded = false;
    this.enableNavMenuClickOutsideDetection = false;
    this.hideAllSubMenus();
    if (sm && sm.url) {
      this.router.navigate([sm.url], { queryParamsHandling: 'merge' });
    }
  }

  onClickMenu(sm) {
    if (sm && sm.url) {
      this.hamburgerExpanded = false;
      this.enableNavMenuClickOutsideDetection = false;
      this.router.navigate([sm.url], { queryParamsHandling: 'merge' });
    }
  }

  onClickedOutsideNavMenu() {
    if (this.enableNavMenuClickOutsideDetection) {
      this.enableNavMenuClickOutsideDetection = false;
      this.hamburgerExpanded = false;
    }
  }

  onClickSearchBox() {
    this.focused = true;
    setTimeout(() => {
      this.searchbox.nativeElement.select();
    });
  }

  onClickedOutsideSearchBox() {
    this.focused = false;
  }

  onClickSearchSuggestion(suggestion) {
    setTimeout(() => {
      this.focused = false;
      this.router.navigate([suggestion.route], {
        queryParamsHandling: 'merge',
      });
    });
  }

  onSearchBoxEnter(input) {
    // navigate to it
    this.focused = false;
    if (
      this.suggestions.length &&
      this.selectedSuggestionIndex < this.suggestions.length
    ) {
      input = this.suggestions[this.selectedSuggestionIndex].route;
    }
    this.router.navigate(input.split(' '), { queryParamsHandling: 'merge' });
  }

  updateSuggestions(input) {
    if (input) {
      const x = input.toLowerCase();
      const sugg = this.knownSuggestions.filter(s => {
        return s.name.toLowerCase().indexOf(x) === 0;
      });
      this.suggestions = sugg;
      this.selectedSuggestionIndex = 0;
    } else {
      this.selectedSuggestionIndex = 0;
      this.suggestions = [];
    }
  }
  onSearchBoxKey(event) {
    const keyCode = event.keyCode;
    switch (keyCode) {
      case KEY_RETURN:
      case KEY_ENTER:
        break;
      case KEY_UP:
        if (this.selectedSuggestionIndex > 0) {
          this.selectedSuggestionIndex--;
        }
        break;
      case KEY_DOWN:
        if (this.selectedSuggestionIndex < this.suggestions.length - 1) {
          this.selectedSuggestionIndex++;
        }
        break;
      default:
        this.updateSuggestions(event.target.value);
        break;
    }
  }

  onMouseOverSuggestion(suggestion, index) {
    this.selectedSuggestionIndex = index;
  }

  onClickLogo() {
    // toggle high contrast mode
    document.body.classList.toggle('high-contrast');
  }

  onClickLogout() {
    localStorage.removeItem('sherlock_creds');
    localStorage.removeItem('sherlock_auth_token');
    localStorage.removeItem('sherlock_refresh_token');
    localStorage.removeItem('sherlock_mynutanix_email');
    localStorage.removeItem('sherlock_role');
    this.authService.setUser('');
    this.router.navigate(['login']);
  }

  checkCurrPassword(ps) {
    const userObj = JSON.parse(localStorage['sherlock_creds']);
    const password = userObj.password;
    if (ps !== password) {
      this.currPsMatch = false;
    } else {
      this.currPsMatch = true;
    }
  }

  checkNewPassword(n, c) {
    if (n === '') {
      this.newPasswordEmpty = true;
    } else {
      this.newPasswordEmpty = false;
    }

    if (n === c) {
      this.newPasswordMatch = true;
    } else {
      this.newPasswordMatch = false;
    }
  }

  onChangePassWord() {
    this.changePasswordExpanded = true;
    this.btnDisabled = false;
  }

  OnTogglePasswordDialog() {
    this.changePasswordExpanded = false;
    return;
  }

  passwordChangeDisabled() {
    return (
      !this.currPsMatch ||
      !this.newPs ||
      !this.newPasswordMatch ||
      this.btnDisabled
    );
  }

  clickChangePassword(changePassword) {
    if (!changePassword) {
      this.changePasswordExpanded = false;
      return;
    }
    this.btnDisabled = true;
    const userObj = JSON.parse(localStorage['sherlock_creds']);
    const userName = userObj.username;
    this.http
      .get('/v1/users', getHttpRequestOptions())
      .toPromise()
      .then(
        res => {
          let user = res.json().find(r => r.email === userName);
          user.password = this.newPs;
          this.http
            .put('/v1/users', user, getHttpRequestOptions())
            .toPromise()
            .then(
              res => {
                this.changePasswordSuccess = true;
                const self = this;
                setTimeout(function() {
                  self.changePasswordExpanded = false;
                  this.changePasswordSuccess = false;
                  self.onClickLogout();
                }, 2000);
              },
              rej => {
                this.changePasswordFail = true;
                const self = this;
                setTimeout(function() {
                  this.changePasswordExpanded = false;
                  self.changePasswordFail = false;
                }, 2000);
              }
            );
        },

        rej => {
          this.changePasswordFail = true;
          const self = this;
          setTimeout(function() {
            this.changePasswordExpanded = false;
            self.changePasswordFail = false;
          }, 2000);
        }
      );
  }

  ngOnDestroy() {
    this.sub.unsubscribe();
    super.ngOnDestroy();
  }
}
