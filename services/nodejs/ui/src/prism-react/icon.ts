import { Component, Input, ElementRef, AfterViewInit } from '@angular/core';
import {
  AlertIcon,
  AnalyzeIcon,
  AnomalyIcon,
  BugIcon,
  CalendarIcon,
  CheckBoxIcon,
  CheckMarkIcon,
  ChevronDownIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
  ChevronUpIcon,
  ClockIcon,
  CloneIcon,
  CloseIcon,
  ColorByIcon,
  ConsoleIcon,
  DashboardIcon,
  DateIcon,
  DotIcon,
  DoubleDotVerticalIcon,
  EditIcon,
  EjectIcon,
  EmptyStarIcon,
  ExclamationIcon,
  ExclamationMarkIcon,
  ExportIcon,
  FilterIcon,
  GearIcon,
  GroupByIcon,
  HealthIcon,
  ImportIcon,
  InfoTooltipIcon,
  KeyboardIcon,
  LabelIcon,
  MagGlassIcon,
  MenuIcon,
  MigrateIcon,
  MinusIcon,
  NutanixLogoIcon,
  OpenInNewWindowIcon,
  PencilIcon,
  PlusIcon,
  PowerIcon,
  QuarantineIcon,
  QuestionIcon,
  QuestionMarkIcon,
  QuestionTooltipIcon,
  RestartIcon,
  RoundedPlusIcon,
  RunIcon,
  SettingsIcon,
  SnapshotIcon,
  SortByIcon,
  StarIcon,
  StarSketchIcon,
  SuspendIcon,
  TaskIcon,
  TimeIcon,
  ToolIcon,
  TriangleDownIcon,
  TriangleLeftIcon,
  TriangleRightIcon,
  TriangleUpDownIcon,
  TriangleUpIcon,
  TripleDotVerticalIcon,
  UploadIcon,
  ZoomInIcon,
  ZoomOutIcon,
} from 'prism-reactjs';
import { PrismReactComponentBase, PrismReactComponentTemplate } from './common';
import { PrismReactService } from './service';

const selector = 'app-prism-react-icon';

@Component({
  selector,
  template: PrismReactComponentTemplate,
})
export class PrismReactIconComponent extends PrismReactComponentBase
  implements AfterViewInit {
  @Input() type: string;
  @Input() color: string;
  @Input() size: string;
  @Input() className: string;

  constructor(prismReactService: PrismReactService, elRef: ElementRef) {
    super(prismReactService, elRef, null);
  }

  isMounted(): boolean {
    return super.isMounted() && this.reactComponent != null;
  }

  ngAfterViewInit() {
    this.initReactComponent();
    this.render();
  }

  initReactComponent() {
    switch (this.type) {
      case 'alert':
        this.reactComponent = AlertIcon;
        break;
      case 'analyze':
        this.reactComponent = AnalyzeIcon;
        break;
      case 'anomaly':
        this.reactComponent = AnomalyIcon;
        break;
      case 'bug':
        this.reactComponent = BugIcon;
        break;
      case 'calendar':
        this.reactComponent = CalendarIcon;
        break;
      case 'checkbox':
        this.reactComponent = CheckBoxIcon;
        break;
      case 'checkmark':
        this.reactComponent = CheckMarkIcon;
        break;
      case 'chevrondown':
        this.reactComponent = ChevronDownIcon;
        break;
      case 'chevronleft':
        this.reactComponent = ChevronLeftIcon;
        break;
      case 'chevronright':
        this.reactComponent = ChevronRightIcon;
        break;
      case 'chevronup':
        this.reactComponent = ChevronUpIcon;
        break;
      case 'clock':
        this.reactComponent = ClockIcon;
        break;
      case 'clone':
        this.reactComponent = CloneIcon;
        break;
      case 'close':
        this.reactComponent = CloseIcon;
        break;
      case 'colorby':
        this.reactComponent = ColorByIcon;
        break;
      case 'console':
        this.reactComponent = ConsoleIcon;
        break;
      case 'dashboard':
        this.reactComponent = DashboardIcon;
        break;
      case 'date':
        this.reactComponent = DateIcon;
        break;
      case 'dot':
        this.reactComponent = DotIcon;
        break;
      case 'doubledot':
        this.reactComponent = DoubleDotVerticalIcon;
        break;
      case 'edit':
        this.reactComponent = EditIcon;
        break;
      case 'eject':
        this.reactComponent = EjectIcon;
        break;
      case 'emptystar':
        this.reactComponent = EmptyStarIcon;
        break;
      case 'exclamation':
        this.reactComponent = ExclamationIcon;
        break;
      case 'exclamationmark':
        this.reactComponent = ExclamationMarkIcon;
        break;
      case 'export':
        this.reactComponent = ExportIcon;
        break;
      case 'filter':
        this.reactComponent = FilterIcon;
        break;
      case 'gear':
        this.reactComponent = GearIcon;
        break;
      case 'groupby':
        this.reactComponent = GroupByIcon;
        break;
      case 'health':
        this.reactComponent = HealthIcon;
        break;
      case 'import':
        this.reactComponent = ImportIcon;
        break;
      case 'infotooltip':
        this.reactComponent = InfoTooltipIcon;
        break;
      case 'keyboard':
        this.reactComponent = KeyboardIcon;
        break;
      case 'label':
        this.reactComponent = LabelIcon;
        break;
      case 'magglass':
        this.reactComponent = MagGlassIcon;
        break;
      case 'menu':
        this.reactComponent = MenuIcon;
        break;
      case 'migrate':
        this.reactComponent = MigrateIcon;
        break;
      case 'minus':
        this.reactComponent = MinusIcon;
        break;
      case 'nutanixlogo':
        this.reactComponent = NutanixLogoIcon;
        break;
      case 'openinnewwindow':
        this.reactComponent = OpenInNewWindowIcon;
        break;
      case 'pencil':
        this.reactComponent = PencilIcon;
        break;
      case 'plus':
        this.reactComponent = PlusIcon;
        break;
      case 'power':
        this.reactComponent = PowerIcon;
        break;
      case 'quarantine':
        this.reactComponent = QuarantineIcon;
        break;
      case 'question':
        this.reactComponent = QuestionIcon;
        break;
      case 'questionmark':
        this.reactComponent = QuestionMarkIcon;
        break;
      case 'questiontooltip':
        this.reactComponent = QuestionTooltipIcon;
        break;
      case 'restart':
        this.reactComponent = RestartIcon;
        break;
      case 'roundedplus':
        this.reactComponent = RoundedPlusIcon;
        break;
      case 'run':
        this.reactComponent = RunIcon;
        break;
      case 'settings':
        this.reactComponent = SettingsIcon;
        break;
      case 'snapshot':
        this.reactComponent = SnapshotIcon;
        break;
      case 'sortby':
        this.reactComponent = SortByIcon;
        break;
      case 'star':
        this.reactComponent = StarIcon;
        break;
      case 'starsketch':
        this.reactComponent = StarSketchIcon;
        break;
      case 'suspend':
        this.reactComponent = SuspendIcon;
        break;
      case 'task':
        this.reactComponent = TaskIcon;
        break;
      case 'time':
        this.reactComponent = TimeIcon;
        break;
      case 'tool':
        this.reactComponent = ToolIcon;
        break;
      case 'triangledown':
        this.reactComponent = TriangleDownIcon;
        break;
      case 'triangleleft':
        this.reactComponent = TriangleLeftIcon;
        break;
      case 'triangleright':
        this.reactComponent = TriangleRightIcon;
        break;
      case 'triangleupdown':
        this.reactComponent = TriangleUpDownIcon;
        break;
      case 'triangleup':
        this.reactComponent = TriangleUpIcon;
        break;
      case 'tripledotvertical':
        this.reactComponent = TripleDotVerticalIcon;
        break;
      case 'upload':
        this.reactComponent = UploadIcon;
        break;
      case 'zoomin':
        this.reactComponent = ZoomInIcon;
        break;
      case 'zoomout':
        this.reactComponent = ZoomOutIcon;
        break;
      default:
        break;
    }
  }

  protected getProps(): any {
    const { color, size, className } = this;
    return {
      color,
      size,
      className,
    };
  }
}
