import {
  Component,
  OnInit,
  ViewChild,
  ElementRef,
  AfterViewInit,
  Input,
  Output,
  forwardRef,
  EventEmitter,
} from '@angular/core';
import { NG_VALUE_ACCESSOR } from '@angular/forms';

declare const monaco: any;
declare const require: any;

@Component({
  selector: 'monaco-editor',
  templateUrl: './monaco-editor.component.html',
  styleUrls: ['./monaco-editor.component.css'],
  providers: [
    {
      provide: NG_VALUE_ACCESSOR,
      useExisting: forwardRef(() => MonacoEditorComponent),
      multi: true,
    },
  ],
})
export class MonacoEditorComponent implements OnInit, AfterViewInit {
  @ViewChild('editor') editorContent: ElementRef;
  @Input() language: string;
  @Input() readOnly: boolean;
  @Input() showDiff: boolean = false;
  @Input() editorWidth: number = 800;
  @Input() editorHeight: number = 600;
  @Input() language_defaults: any = null;
  @Input() options: any = {};
  @Input()
  set value(v: string | object) {
    if (v !== this._value) {
      this._value = v;
      this.onChange(v);
    }
  }
  @Output() change = new EventEmitter();
  @Output() instance = null;

  private _editor: any;
  private _value = null;
  private _javascriptExtraLibs: any = null;
  private _typescriptExtraLibs: any = null;
  private _originalValue = '';
  private _modifiedValue = '';

  constructor() {}

  get value(): string | object {
    return this._value;
  }

  ngOnInit() {}

  ngAfterViewInit() {
    var onGotAmdLoader = () => {
      // Load monaco
      (<any>window).require.config({ paths: { vs: 'assets/monaco/vs' } });
      (<any>window).require(['vs/editor/editor.main'], () => {
        setTimeout(() => {
          this.initMonaco();
        });
      });
    };

    // Load AMD loader if necessary
    if (!(<any>window).require) {
      var loaderScript = document.createElement('script');
      loaderScript.type = 'text/javascript';
      loaderScript.src = 'assets/monaco/vs/loader.js';
      loaderScript.addEventListener('load', onGotAmdLoader);
      document.body.appendChild(loaderScript);
    } else {
      onGotAmdLoader();
    }
  }

  /**
   * Upon destruction of the component we make sure to dispose both the editor and the extra libs that we might've loaded
   */
  ngOnDestroy() {
    this._editor.dispose();
    if (this._javascriptExtraLibs !== null) {
      this._javascriptExtraLibs.dispose();
    }

    if (this._typescriptExtraLibs !== null) {
      this._typescriptExtraLibs.dispose();
    }
  }

  // Will be called once monaco library is available
  initMonaco() {
    var myDiv: HTMLDivElement = this.editorContent.nativeElement;
    // instead of requiring caller to pass in precise editorWidth, editorHeight,
    // we just size to 100% of our container size, so caller just need to
    // style the parent container properly
    myDiv.style.width = '100%'; //this.editorWidth + 'px';
    myDiv.style.height = '100%'; //this.editorHeight + 'px';

    let options = this.options;
    options.language = this.language;
    options.readOnly = this.readOnly || false;

    if (!this.showDiff) {
      options.value = this._value;
      this._editor = monaco.editor.create(myDiv, options);
      this._editor.getModel().onDidChangeContent(e => {
        this.updateValue(this._editor.getModel().getValue());
      });
    } else {
      options.enableSplitViewResizing = false;
      // options.readOnly = true;
      this._editor = monaco.editor.createDiffEditor(myDiv, options);
      var originalModel = monaco.editor.createModel(
        this._originalValue,
        'text/' + this.language
      );
      var modifiedModel = monaco.editor.createModel(
        this._modifiedValue,
        'text/' + +this.language
      );
      this._editor.setModel({
        original: originalModel,
        modified: modifiedModel,
      });
      this._editor
        .getModifiedEditor()
        .getModel()
        .onDidChangeContent(e => {
          var modifiedValue = this._editor.getModifiedEditor().getValue();
          this.updateValue(Object.assign(this._value, { modifiedValue }));
        });
    }

    // Set language defaults
    // We already set the language on the component so we act accordingly
    if (this.language_defaults !== null) {
      switch (this.language) {
        case 'javascript':
          monaco.languages.typescript.javascriptDefaults.setCompilerOptions(
            this.language_defaults.compilerOptions
          );
          for (var extraLib in this.language_defaults.extraLibs) {
            this._javascriptExtraLibs = monaco.languages.typescript.javascriptDefaults.addExtraLib(
              this.language_defaults.extraLibs[extraLib].definitions,
              this.language_defaults.extraLibs[extraLib].definitions_name
            );
          }
          break;
        case 'typescript':
          monaco.languages.typescript.typescriptDefaults.setCompilerOptions(
            this.language_defaults.compilerOptions
          );
          for (var extraLib in this.language_defaults.extraLibs) {
            this._typescriptExtraLibs = monaco.languages.typescript.typescriptDefaults.addExtraLib(
              this.language_defaults.extraLibs[extraLib].definitions,
              this.language_defaults.extraLibs[extraLib].definitions_name
            );
          }
          break;
      }
    }

    // Currently setting this option prevents the autocomplete selection with the "Enter" key
    // TODO make sure to propagate the event to the autocomplete
    if (this.options.customPreventCarriageReturn === true) {
      let preventCarriageReturn = this._editor.addCommand(
        monaco.KeyCode.Enter,
        function() {
          return false;
        }
      );
    }
  }

  /**
   * UpdateValue
   *
   * @param value
   */
  updateValue(value: string | object) {
    this.value = value;
    this.onChange(value);
    this.onTouched();
    this.change.emit(value);
  }

  /**
   * WriteValue
   * Implements ControlValueAccessor
   *
   * @param value
   */
  writeValue(value: string | object) {
    this._value = value || '';
    if (value && typeof value === 'object') {
      if ('originalValue' in value) {
        this._originalValue = (value as any).originalValue;
      }
      if ('modifiedValue' in value) {
        this._modifiedValue = (value as any).modifiedValue;
      }
    }
    if (this.instance) {
      this.instance.setValue(this._value);
    }
    // If an instance of Monaco editor is running, update its contents
    if (!this.showDiff && this._editor) {
      this._editor.getModel().setValue(this._value);
    } else if (this._editor) {
      this._editor
        .getOriginalEditor()
        .getModel()
        .setValue(this._originalValue);
      this._editor
        .getModifiedEditor()
        .getModel()
        .setValue(this._modifiedValue);
    }
  }

  onChange(_) {}
  onTouched() {}
  registerOnChange(fn) {
    this.onChange = fn;
  }
  registerOnTouched(fn) {
    this.onTouched = fn;
  }

  renderSideBySide(value: boolean) {
    this._editor.updateOptions({
      renderSideBySide: value,
    });
  }
}
