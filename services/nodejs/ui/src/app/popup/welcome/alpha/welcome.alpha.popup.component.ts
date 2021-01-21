import { Component } from '@angular/core';
import { Router } from '@angular/router';
import { OnBoardService } from '../../../services/onboard.service';
import { AuthService } from '../../../guards/auth.service';

@Component({
  selector: 'app-welcome-alpha-popup',
  templateUrl: './welcome.alpha.popup.component.html',
  styleUrls: ['./welcome.alpha.popup.component.css'],
})
export class WelcomeAlphaPopupComponent {
  constructor(
    private router: Router,
    private onboardService: OnBoardService,
    private authService: AuthService
  ) {}
  onClosePopup() {
    this.onboardService.setOnBoarded(this.authService.getUser(), true);
    this.router.navigate([{ outlets: { popup: null } }]);
  }
}
