import { BrowserModule } from '@angular/platform-browser';
import { BrowserAnimationsModule } from '@angular/platform-browser/animations';
import { NgModule } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { MonacoEditorComponent } from '../component-library/ng2-monaco-editor';
import { ClickOutsideModule } from 'ng-click-outside';

import { Router } from '@angular/router';
import { HttpModule, Http, XHRBackend, RequestOptions } from '@angular/http';
import { httpFactory } from './mock/mock.http';

import { NgZorroAntdModule } from 'ng-zorro-antd';

import { StorageCapacityPipe, StringArrayPipe } from './pipes/index';

import { AppComponent } from './app.component';
import { AppHeaderComponent } from './header/app.header.component';
import { LoginComponent } from './login/login.component';
import { MyNutanixComponent } from './mynutanix/mynutanix.component';
import { DashboardComponent } from './dashboard/dashboard.component';
import {
  ProjectsComponent,
  ProjectsListComponent,
  ProjectsSummaryComponent,
  ProjectsAlertsComponent,
} from './projects';
import {
  ProjectComponent,
  ProjectSummaryComponent,
  ProjectAlertsComponent,
  ProjectApplicationsComponent,
  ProjectDatastreamsComponent,
  ProjectDatasourcesComponent,
  ProjectScriptsComponent,
  ProjectEdgesComponent,
  ProjectUsersComponent,
  ProjectRuntimeComponent,
} from './projects/project/index';

import { ProjectPopupComponent } from './popup/project/project.popup.component';
import { ProjectDataStreamPopupComponent } from './popup/project/data-stream/project.data-stream.popup.component';
import { LogComponent } from './log/log.component';
import {
  ApplicationsComponent,
  ApplicationComponent,
  ApplicationAlertsComponent,
  ApplicationDeploymentComponent,
  ApplicationSummaryComponent,
  ApplicationLogsComponent,
} from './applications/index';
import {
  ApplicationsPopupComponent,
  ApplicationsCreateApplicationPopupComponent,
} from './popup/applications/index';
import { UsersComponent } from './users/users.component';
import { OtaComponent } from './ota/ota.component';
import { OtaUpdateComponent } from './ota/update/ota.update.component';
import { OtaInProgressComponent } from './ota/inProgress/ota.inProgress.component';
import {
  OtaPopupComponent,
  OtaConfirmUpdatePopupComponent,
} from './popup/ota/index';
import {
  CategoriesPopupComponent,
  CategoriesCreateCategoryPopupComponent,
} from './popup/categories/index';

import { AppRoutingModule } from './app-routing.module';

import { AuthGuard } from './guards/auth.guard';
import { AuthService } from './guards/auth.service';
import { RegistryService } from './services/registry.service';
import { LoginService } from './login/login.service';
import { OnBoardService } from './services/onboard.service';

import 'rxjs/add/operator/toPromise';
import {
  EdgesComponent,
  EdgeComponent,
  EdgeAlertsComponent,
  EdgeDataSourcesComponent,
  EdgeEdgeDevicesComponent,
  EdgeMetricsComponent,
  EdgeSettingsComponent,
  EdgeSummaryComponent,
  EdgePopupComponent,
  EdgeCreateDataSourcePopupComponent,
} from './edges/index';

import {
  CategoriesComponent,
  CategoryComponent,
  CategoryValuesComponent,
} from './categories';

import { ScriptsComponent } from './scripts/index';

import { ScriptsRuntimeComponent } from './runtime/index';

import {
  DataStreamsComponent,
  DataStreamsSummaryComponent,
  DataStreamsListComponent,
  DatastreamsVisualizationComponent,
  DataStreamsAlertsComponent,
  DataStreamsMetricsComponent,
} from './datastreams/index';

import {
  DataSourcesComponent,
  DataSourcesAlertsComponent,
  DataSourcesListComponent,
  DataSourcesSummaryComponent,
} from './datasources/index';

import { SettingsComponent, GeneralComponent } from './settings/index';
import { ContainerComponent } from './container/index';

import { CloudsComponent } from './clouds/index';

import {
  ScriptsPopupComponent,
  ScriptsUploadPopupComponent,
  ScriptsEditPopupComponent,
  ScriptsCreateRuntimePopupComponent,
} from './popup/scripts/index';

import {
  DataSourcesPopupComponent,
  DataSourcesCreateDataSourcePopupComponent,
} from './popup/datasources/index';

import {
  ContainerPopupComponent,
  ContainerCreateContainerPopupComponent,
} from './popup/container/index';

import {
  ProjectsPopupComponent,
  ProjectsCreateProjectPopupComponent,
} from './popup/projects/index';

import {
  DataStreamsPopupComponent,
  DataStreamsCreateDataStreamPopupComponent,
} from './popup/datastreams/index';

import {
  EdgesPopupComponent,
  EdgesCreateEdgePopupComponent,
} from './popup/edges/index';

import { ComponentsCreateContainerPopupComponent } from './popup/components/create-container/index';

import {
  WelcomePopupComponent,
  WelcomeAlphaPopupComponent,
} from './popup/welcome/index';

import {
  PrismReactButtonComponent,
  PrismReactParagraphComponent,
  PrismReactInputPlusLabelComponent,
  PrismReactModalComponent,
  PrismReactAlertComponent,
  PrismReactBadgeComponent,
  PrismReactStatusIconComponent,
  PrismReactTextLabelComponent,
  PrismReactTableComponent,
  PrismReactAreaChartComponent,
  PrismReactBarChartComponent,
  PrismReactBarChartBillingComponent,
  PrismReactDistributionBarChartComponent,
  PrismReactDonutChartComponent,
  PrismReactLineChartComponent,
  PrismReactPieChartComponent,
  PrismReactSparkLineComponent,
  PrismReactViolationChartComponent,
  PrismReactDatePickerComponent,
  PrismReactInputFileUploadComponent,
  PrismReactIconComponent,
} from '../prism-react/index';
import { PrismReactService } from '../prism-react/service';

import { PrismReactTestComponent } from './prism-react-test/prism-react-test.component';

let isMock = false;
if (location.search) {
  isMock = location.search
    .substring(1)
    .split('&')
    .some(x => {
      const ts = x.split('=');
      return ts.length === 2 && ts[0] === 'model' && ts[1] === 'mock';
    });
}

const providers: any[] = [
  AuthGuard,
  AuthService,
  RegistryService,
  LoginService,
  OnBoardService,
  PrismReactService,
];
if (isMock) {
  providers.push({
    provide: Http,
    useFactory: httpFactory,
    deps: [XHRBackend, RequestOptions],
  });
}

@NgModule({
  providers,
  declarations: [
    MonacoEditorComponent,
    StorageCapacityPipe,
    StringArrayPipe,
    AppComponent,
    AppHeaderComponent,
    LoginComponent,
    MyNutanixComponent,
    DashboardComponent,
    ProjectsComponent,
    ProjectsListComponent,
    ProjectsSummaryComponent,
    ProjectsAlertsComponent,
    ProjectsPopupComponent,
    ProjectsCreateProjectPopupComponent,
    EdgesComponent,
    EdgesPopupComponent,
    EdgesCreateEdgePopupComponent,
    EdgeComponent,
    EdgeAlertsComponent,
    EdgeDataSourcesComponent,
    EdgeEdgeDevicesComponent,
    EdgeMetricsComponent,
    EdgeSettingsComponent,
    EdgeSummaryComponent,
    EdgePopupComponent,
    EdgeCreateDataSourcePopupComponent,
    ProjectComponent,
    ProjectSummaryComponent,
    ProjectAlertsComponent,
    ProjectApplicationsComponent,
    ProjectDatastreamsComponent,
    ProjectScriptsComponent,
    ProjectRuntimeComponent,
    ProjectEdgesComponent,
    ProjectDatasourcesComponent,
    ProjectUsersComponent,
    ProjectPopupComponent,
    ProjectDataStreamPopupComponent,
    ScriptsComponent,
    ScriptsRuntimeComponent,
    CategoriesComponent,
    CategoryComponent,
    CategoryValuesComponent,
    DataStreamsComponent,
    DataStreamsSummaryComponent,
    DataStreamsListComponent,
    DatastreamsVisualizationComponent,
    DataStreamsAlertsComponent,
    DataStreamsMetricsComponent,
    DataSourcesComponent,
    DataSourcesAlertsComponent,
    DataSourcesSummaryComponent,
    DataSourcesListComponent,
    CategoriesPopupComponent,
    CategoriesCreateCategoryPopupComponent,
    ScriptsPopupComponent,
    ScriptsUploadPopupComponent,
    ScriptsEditPopupComponent,
    ScriptsCreateRuntimePopupComponent,
    DataSourcesPopupComponent,
    DataSourcesCreateDataSourcePopupComponent,
    DataStreamsPopupComponent,
    DataStreamsCreateDataStreamPopupComponent,
    CloudsComponent,
    LogComponent,
    ApplicationsPopupComponent,
    ApplicationsCreateApplicationPopupComponent,
    ApplicationsComponent,
    ApplicationAlertsComponent,
    ApplicationComponent,
    ApplicationDeploymentComponent,
    ApplicationSummaryComponent,
    ApplicationLogsComponent,
    UsersComponent,
    OtaComponent,
    OtaInProgressComponent,
    OtaUpdateComponent,
    OtaPopupComponent,
    OtaConfirmUpdatePopupComponent,
    WelcomePopupComponent,
    WelcomeAlphaPopupComponent,
    SettingsComponent,
    ContainerComponent,
    GeneralComponent,
    ContainerPopupComponent,
    ContainerCreateContainerPopupComponent,
    ComponentsCreateContainerPopupComponent,
    PrismReactButtonComponent,
    PrismReactParagraphComponent,
    PrismReactInputPlusLabelComponent,
    PrismReactModalComponent,
    PrismReactTestComponent,
    PrismReactAlertComponent,
    PrismReactBadgeComponent,
    PrismReactStatusIconComponent,
    PrismReactTextLabelComponent,
    PrismReactTableComponent,
    PrismReactAreaChartComponent,
    PrismReactBarChartComponent,
    PrismReactBarChartBillingComponent,
    PrismReactDistributionBarChartComponent,
    PrismReactDonutChartComponent,
    PrismReactLineChartComponent,
    PrismReactPieChartComponent,
    PrismReactSparkLineComponent,
    PrismReactViolationChartComponent,
    PrismReactDatePickerComponent,
    PrismReactInputFileUploadComponent,
    PrismReactIconComponent,
  ],
  imports: [
    BrowserModule,
    FormsModule,
    HttpModule,
    BrowserAnimationsModule,
    NgZorroAntdModule.forRoot(),
    AppRoutingModule,
    ClickOutsideModule,
  ],
  bootstrap: [AppComponent],
})
export class AppModule {
  // Diagnostic only: inspect router configuration
  constructor(router: Router) {
    console.log('Routes: ', JSON.stringify(router.config, undefined, 2));
  }
}
