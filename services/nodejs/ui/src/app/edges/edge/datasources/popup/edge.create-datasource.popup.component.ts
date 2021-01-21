import { Component } from '@angular/core';
import { Router } from '@angular/router';

@Component({
  selector: 'app-edge-create-datasource-popup',
  templateUrl: './edge.create-datasource.popup.component.html',
  styleUrls: ['./edge.create-datasource.popup.component.css'],
})
export class EdgeCreateDataSourcePopupComponent {
  constructor(private router: Router) {}

  isVisible = true;
  isConfirmLoading = false;

  sensorType = 'Sensor';
  securityType = 'Secure';

  current = 0;

  okBtnText = 'Next';
  cancelBtnText = 'Cancel';

  handleOk = e => {
    if (this.current === 2) {
      this.onCreateDataSource();
    } else if (this.current === 1) {
      this.current = 2;
      this.okBtnText = 'Create';
    } else {
      this.current = 1;
    }
  };

  handleCancel = e => {
    this.isVisible = false;
    this.onClosePopup();
  };

  handleBack = e => {
    this.current--;
    this.okBtnText = 'Next';
  };

  onClosePopup() {
    this.router.navigate([{ outlets: { popup: null } }]);
  }

  onCreateDataSource() {
    this.isConfirmLoading = true;
    setTimeout(() => {
      this.isVisible = false;
      this.isConfirmLoading = false;
      this.router.navigate([{ outlets: { popup: null } }]);
    }, 3000);
  }
}
