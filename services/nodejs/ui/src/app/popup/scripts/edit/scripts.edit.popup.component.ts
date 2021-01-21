import { Component, OnDestroy, OnInit, ViewChild } from '@angular/core';
import { Router, ActivatedRoute } from '@angular/router';
import { Http } from '@angular/http';
import { RegistryService } from '../../../services/registry.service';
import { getHttpRequestOptions } from '../../../utils/httpUtil';
import { handleAuthError } from '../../../utils/authUtil';

function scriptTypeToRadioValue(type: string): string {
  return type === 'Transformation' ? 'transformation' : 'lambda';
}
function radioValueToScriptType(radioValue: string): string {
  return radioValue === 'transformation' ? 'Transformation' : 'Function';
}
@Component({
  selector: 'app-scripts-edit-popup',
  templateUrl: './scripts.edit.popup.component.html',
  styleUrls: ['./scripts.edit.popup.component.css'],
})
export class ScriptsEditPopupComponent implements OnInit, OnDestroy {
  sub = null;
  script = null;
  isConfirmLoading = false;
  code = '';
  canDiff = false;
  showDiff = false;
  inline = false;
  diffObj = {
    originalValue: '',
    modifiedValue: '',
  };

  languages = ['go', 'javascript', 'python', 'ruby', 'php'];
  language = 'javascript';
  radioValue = 'lambda';
  selectedTab = 'script';
  themes = [
    {
      name: 'Normal',
      value: 'vs',
    },
    {
      name: 'Dark',
      value: 'vs-dark',
    },
    {
      name: 'High Contrast Dark',
      value: 'hc-black',
    },
  ];
  theme = 'vs';
  uploadFile = false;
  file = null;

  // which step are we on in the create flow
  current = 0;

  @ViewChild('monacoDiffEditor') monacoDiffEditor;

  constructor(
    private route: ActivatedRoute,
    private router: Router,
    private http: Http,
    private regService: RegistryService
  ) {}

  onClosePopup() {
    this.router.navigate([{ outlets: { popup2: null } }]);
  }

  isUpdateDisabled() {
    return false;
  }

  onUpdateScript() {
    this.isConfirmLoading = true;
    if (this.uploadFile) {
      this.handleFileRead();
      return;
    }
    // Handle edit mode
    if (this.showDiff) {
      // check if modified is equal as code if not then set it as code as user
      // might have updated it in diff editor
      if (this.diffObj.modifiedValue !== this.code) {
        this.code = this.diffObj.modifiedValue;
      }
    }
    this.script.code = this.code;

    this.updateScript();
  }

  handleFileRead() {
    if (!this.file) {
      alert('Please select file');
      this.isConfirmLoading = false;
      return;
    }
    const reader = new FileReader();
    reader.onload = e => {
      this.script.code = e.target['result'];
      this.updateScript();
    };
    reader.readAsText(this.file);
  }

  updateScript() {
    this.script.language = this.language;
    this.script.type = radioValueToScriptType(this.radioValue);
    this.http
      .put('/v1/scripts', this.script, getHttpRequestOptions())
      .toPromise()
      .then(
        r => {
          this.isConfirmLoading = false;
          this.onClosePopup();
        },
        e => {
          this.isConfirmLoading = false;
          handleAuthError(null, e, this.router, this.http, () =>
            this.updateScript()
          );
        }
      );
  }

  handleFileSelect(event) {
    if (event.target.files.length) {
      this.file = event.target.files[0];
    } else {
      this.file = null;
    }
  }

  onChangeTheme() {
    if (window['monaco']['editor']) {
      window['monaco']['editor'].setTheme(this.theme);
    }
  }

  ngOnInit() {
    this.sub = this.route.params.subscribe(params => {
      this.script = this.regService.get(params.id);
      this.code = this.script.code;
      if (this.script.language) {
        this.language = this.script.language;
      }
      this.radioValue = scriptTypeToRadioValue(this.script.type);
    });
  }

  ngOnDestroy() {
    if (this.script) {
      this.regService.register(this.script.id, null);
    }
    this.sub.unsubscribe();
  }
  /**
   * Model change handler
   * Enable/disable show diff option
   */
  updateDiffOption() {
    if (this.code === this.script.code) {
      this.canDiff = false;
    } else {
      this.canDiff = true;
    }
  }
  /**
   * Show diff handler
   * Set the diff model object and show diff editor
   */
  displayDiffEditor() {
    // set diffObj for diff editor before making it visible
    this.showDiff = !this.showDiff;
    if (this.showDiff) {
      this.diffObj.originalValue = this.script.code;
      this.diffObj.modifiedValue = this.code;
    } else {
      // set value form diff editor incase user modified it there.
      this.code = this.diffObj.modifiedValue;
    }
  }
  /**
   * Toggle inline diff option handler
   */
  updateDiffRenderOption() {
    this.monacoDiffEditor.renderSideBySide(!this.inline);
  }

  showTab(tabName) {
    this.selectedTab = tabName;
  }

  toggleUploadFile() {
    this.uploadFile = !this.uploadFile;
  }
}
