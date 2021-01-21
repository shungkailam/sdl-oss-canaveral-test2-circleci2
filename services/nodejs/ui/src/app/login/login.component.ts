import { Component, ViewChild, OnInit } from '@angular/core';
import {
  Router,
  CanActivate,
  ActivatedRoute,
  RouterStateSnapshot,
  Params,
} from '@angular/router';
import { AuthService } from '../guards/auth.service';
import { Http } from '@angular/http';
import {
  RegistryService,
  REG_KEY_TENANT_ID,
} from '../services/registry.service';
import { LoginService } from './login.service';

const MOVING_TEXT = 'Freeze space-time continuum!';
const FREEZE_TEXT = 'Engage the warp drive!';

@Component({
  selector: 'app-login',
  templateUrl: './login.component.html',
  styleUrls: ['./login.component.css'],
})
export class LoginComponent implements OnInit {
  returnUrl: string = '';

  username = '';
  password = '';
  moving = true;

  particleUpdateFn = null;

  @ViewChild('usernameEl') usernameElement;
  @ViewChild('passwordEl') passwordElement;

  constructor(
    private router: Router,
    private activatedRoute: ActivatedRoute,
    private authService: AuthService,
    private regService: RegistryService,
    private loginService: LoginService,
    private http: Http
  ) {}

  ngOnInit() {
    // subscribe to router event
    this.activatedRoute.queryParams.subscribe((params: Params) => {
      if (window.location.href.indexOf('login') === -1) {
        this.returnUrl = params['returnUrl'];
      }
    });

    window['particlesJS'].load(
      'particlesGradient',
      'assets/particles.json',
      function() {
        console.log('callback - particles.js config loaded');
      }
    );
  }

  onEnterCommon() {
    if (this.username && this.password) {
      this.loginService
        .login({ email: this.username, password: this.password })
        .then(
          () => {
            const url = this.returnUrl || '';
            if (localStorage['sherlock_role'] !== '')
              this.router.navigate(['edges']);
            else this.router.navigate([url]);
          },
          err => {
            alert("Login failed! Username or password doesn't match!");
            this.password = '';
            this.focusUsername();
          }
        );
    } else if (this.username) {
      this.focusPassword();
    } else {
      this.focusUsername();
    }
  }
  onEnterUsername() {
    this.onEnterCommon();
  }

  onEnterPassword() {
    this.onEnterCommon();
  }

  onClickSubmit(event) {
    event.preventDefault();
    if (!this.username) {
      alert('Please enter username');
      this.focusUsername();
    } else if (!this.password) {
      alert('Please enter password');
      this.focusPassword();
    } else {
      this.onEnterCommon();
    }
  }

  logInMyNutanix() {
    this.returnUrl = this.returnUrl ? this.returnUrl : 'edges';
    window.location.href =
      window.location.protocol +
      '//' +
      window.location.host +
      '/v1/oauth2/authorize?returnUrl=' +
      this.returnUrl;
  }

  focusUsername() {
    this.usernameElement.nativeElement.focus();
  }
  focusPassword() {
    this.passwordElement.nativeElement.focus();
  }

  getVideoLinkText() {
    return this.moving ? MOVING_TEXT : FREEZE_TEXT;
  }

  onClickVideoControl() {
    if (this.moving) {
      this.moving = false;
      this.particleUpdateFn = window['pJSDom'][0].pJS.fn.particlesUpdate;
      window['pJSDom'][0].pJS.fn.particlesUpdate = function() {};
    } else {
      this.moving = true;
      window['pJSDom'][0].pJS.fn.particlesUpdate = this.particleUpdateFn;
    }
  }
}
