import { Component } from '@angular/core';
import { Router } from '@angular/router';

@Component({
  selector: 'app-project-data-stream-popup',
  templateUrl: './project.data-stream.popup.component.html',
  styleUrls: ['./project.data-stream.popup.component.css'],
})
export class ProjectDataStreamPopupComponent {
  constructor(private router: Router) {}

  onClosePopup() {
    this.router.navigate([{ outlets: { popup: null } }]);
  }
}
