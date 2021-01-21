// for new UI change
import { NgModule } from '@angular/core';
import { RouterModule, Routes } from '@angular/router';
import { LoginComponent } from './login/login.component';
import { MyNutanixComponent } from './mynutanix/mynutanix.component';
import { DashboardComponent } from './dashboard/dashboard.component';
import {
  ProjectsComponent,
  ProjectsListComponent,
  ProjectsSummaryComponent,
  ProjectsAlertsComponent,
} from './projects/index';

import { NZ_LOCALE, enUS, NgZorroAntdModule } from 'ng-zorro-antd';

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
} from './categories/index';

import {
  ProjectComponent,
  ProjectSummaryComponent,
  ProjectAlertsComponent,
  ProjectApplicationsComponent,
  ProjectDatastreamsComponent,
  ProjectDatasourcesComponent,
  ProjectScriptsComponent,
  ProjectRuntimeComponent,
  ProjectEdgesComponent,
  ProjectUsersComponent,
} from './projects/project/index';

import { ProjectPopupComponent } from './popup/project/project.popup.component';
import { ProjectDataStreamPopupComponent } from './popup/project/data-stream/project.data-stream.popup.component';
import { UsersComponent } from './users/users.component';
import { OtaComponent } from './ota/ota.component';
import { OtaUpdateComponent } from './ota/update/ota.update.component';
import { OtaInProgressComponent } from './ota/inProgress/ota.inProgress.component';
import {
  OtaPopupComponent,
  OtaConfirmUpdatePopupComponent,
} from './popup/ota/index';
import { AuthGuard } from './guards/auth.guard';
import { AppHeaderComponent } from './header/app.header.component';

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

import {
  ApplicationsComponent,
  ApplicationComponent,
  ApplicationDeploymentComponent,
  ApplicationSummaryComponent,
  ApplicationAlertsComponent,
  ApplicationLogsComponent,
} from './applications/index';

import { SettingsComponent, GeneralComponent } from './settings/index';
import { ContainerComponent } from './container/index';
import { CloudsComponent } from './clouds/index';

import {
  CategoriesPopupComponent,
  CategoriesCreateCategoryPopupComponent,
} from './popup/categories/index';

import {
  ApplicationsPopupComponent,
  ApplicationsCreateApplicationPopupComponent,
} from './popup/applications/index';

import {
  ProjectsPopupComponent,
  ProjectsCreateProjectPopupComponent,
} from './popup/projects/index';

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
  DataStreamsPopupComponent,
  DataStreamsCreateDataStreamPopupComponent,
} from './popup/datastreams/index';

import {
  EdgesPopupComponent,
  EdgesCreateEdgePopupComponent,
} from './popup/edges/index';

import {
  WelcomePopupComponent,
  WelcomeAlphaPopupComponent,
} from './popup/welcome/index';

import { LogComponent } from './log/log.component';

import { PrismReactTestComponent } from './prism-react-test/prism-react-test.component';

const appRoutes: Routes = [
  {
    path: 'login',
    component: LoginComponent,
  },
  {
    path: 'auth/oauth',
    component: MyNutanixComponent,
  },
  {
    path: '',
    component: AppHeaderComponent,
    canActivate: [AuthGuard],
    children: [
      {
        path: 'dashboard',
        component: DashboardComponent,
      },
      {
        path: 'ota',
        component: OtaComponent,
        children: [
          { path: '', redirectTo: 'update', pathMatch: 'full' },
          {
            path: 'update',
            component: OtaUpdateComponent,
          },
          {
            path: 'inprogress',
            component: OtaInProgressComponent,
          },
        ],
      },
      {
        path: 'prism-react-test',
        component: PrismReactTestComponent,
      },
      {
        path: 'projects',
        component: ProjectsComponent,
        children: [
          { path: '', redirectTo: 'list', pathMatch: 'full' },
          {
            path: 'summary',
            component: ProjectsSummaryComponent,
          },
          {
            path: 'list',
            component: ProjectsListComponent,
          },
          {
            path: 'alerts',
            component: ProjectsAlertsComponent,
          },
        ],
      },
      {
        path: 'log',
        component: LogComponent,
      },
      {
        path: 'applications',
        component: ApplicationsComponent,
      },
      {
        path: 'users',
        component: UsersComponent,
      },
      {
        path: 'clouds',
        component: CloudsComponent,
      },
      {
        path: 'container',
        component: ContainerComponent,
      },

      {
        path: 'edges',
        component: EdgesComponent,
      },
      {
        path: 'edge/:id',
        component: EdgeComponent,
        children: [
          { path: '', redirectTo: 'datasources', pathMatch: 'full' },
          {
            path: 'summary',
            component: EdgeSummaryComponent,
          },
          {
            path: 'alerts',
            component: EdgeAlertsComponent,
          },
          {
            path: 'datasources',
            component: EdgeDataSourcesComponent,
          },
          {
            path: 'edgedevices',
            component: EdgeEdgeDevicesComponent,
          },
          {
            path: 'metrics',
            component: EdgeMetricsComponent,
          },
          {
            path: 'settings',
            component: EdgeSettingsComponent,
          },
        ],
      },
      {
        path: 'category/:id',
        component: CategoryComponent,
        children: [
          { path: '', redirectTo: 'category-values', pathMatch: 'full' },
          {
            path: 'category-values',
            component: CategoryValuesComponent,
          },
        ],
      },
      {
        path: 'application/:id',
        component: ApplicationComponent,
        children: [
          { path: '', redirectTo: 'summary', pathMatch: 'full' },
          {
            path: 'summary',
            component: ApplicationSummaryComponent,
          },
          {
            path: 'deployment',
            component: ApplicationDeploymentComponent,
          },
          {
            path: 'alerts',
            component: ApplicationAlertsComponent,
          },
          {
            path: 'logs',
            component: ApplicationLogsComponent,
          },
        ],
      },
      {
        path: 'scripts',
        component: ScriptsComponent,
      },
      {
        path: 'runtime',
        component: ScriptsRuntimeComponent,
      },
      {
        path: 'categories',
        component: CategoriesComponent,
      },
      {
        path: 'datastreams',
        component: DataStreamsComponent,
        children: [
          { path: '', redirectTo: 'list', pathMatch: 'full' },
          {
            path: 'summary',
            component: DataStreamsSummaryComponent,
          },
          {
            path: 'list',
            component: DataStreamsListComponent,
          },
          {
            path: 'visualization',
            component: DatastreamsVisualizationComponent,
          },
          {
            path: 'alerts',
            component: DataStreamsAlertsComponent,
          },
          {
            path: 'metrics',
            component: DataStreamsMetricsComponent,
          },
        ],
      },
      {
        path: 'datasources',
        component: DataSourcesComponent,
        children: [
          { path: '', redirectTo: 'list', pathMatch: 'full' },
          {
            path: 'summary',
            component: DataSourcesSummaryComponent,
          },
          {
            path: 'list',
            component: DataSourcesListComponent,
          },
          {
            path: 'alerts',
            component: DataSourcesAlertsComponent,
          },
        ],
      },
      {
        path: 'settings',
        component: SettingsComponent,
        children: [
          { path: '', redirectTo: 'general', pathMatch: 'full' },
          {
            path: 'general',
            component: GeneralComponent,
          },
        ],
      },
      {
        path: 'project/:id',
        component: ProjectComponent,
        children: [
          { path: '', redirectTo: 'edges', pathMatch: 'full' },
          {
            path: 'summary',
            component: ProjectSummaryComponent,
          },
          {
            path: 'alerts',
            component: ProjectAlertsComponent,
          },
          {
            path: 'applications',
            component: ProjectApplicationsComponent,
          },
          {
            path: 'datastreams',
            component: ProjectDatastreamsComponent,
          },
          {
            path: 'scripts',
            component: ProjectScriptsComponent,
          },
          {
            path: 'runtime',
            component: ProjectRuntimeComponent,
          },
          {
            path: 'edges',
            component: ProjectEdgesComponent,
          },
          {
            path: 'datasources',
            component: ProjectDatasourcesComponent,
          },
          {
            path: 'users',
            component: ProjectUsersComponent,
          },
        ],
      },
      // use relative path here so router will use the query params from the source url
      // see https://angular.io/guide/router
      { path: '', redirectTo: 'projects', pathMatch: 'full' },
      { path: '**', redirectTo: '' },
    ],
  },

  {
    outlet: 'popup',
    path: 'project',
    component: ProjectPopupComponent,
    children: [
      {
        path: 'data-stream',
        component: ProjectDataStreamPopupComponent,
      },
    ],
  },
  {
    outlet: 'popup',
    path: 'ota',
    component: OtaPopupComponent,
    children: [
      {
        path: 'confirm-update',
        component: OtaConfirmUpdatePopupComponent,
      },
    ],
  },
  {
    outlet: 'popup',
    path: 'categories',
    component: CategoriesPopupComponent,
    children: [
      {
        path: 'create-category',
        component: CategoriesCreateCategoryPopupComponent,
      },
    ],
  },
  {
    outlet: 'popup',
    path: 'applications',
    component: ApplicationsPopupComponent,
    children: [
      {
        path: 'create-application',
        component: ApplicationsCreateApplicationPopupComponent,
      },
    ],
  },
  {
    outlet: 'popup',
    path: 'scripts',
    component: ScriptsPopupComponent,
    children: [
      {
        path: 'create-runtime',
        component: ScriptsCreateRuntimePopupComponent,
      },
    ],
  },
  {
    outlet: 'popup2',
    path: 'scripts',
    component: ScriptsPopupComponent,
    children: [
      {
        path: 'upload',
        component: ScriptsUploadPopupComponent,
      },
      {
        path: 'edit/:id',
        component: ScriptsEditPopupComponent,
      },
    ],
  },
  {
    outlet: 'popup',
    path: 'edge',
    component: EdgePopupComponent,
    children: [
      {
        path: 'create-datasource',
        component: EdgeCreateDataSourcePopupComponent,
      },
    ],
  },
  {
    outlet: 'popup',
    path: 'datasources',
    component: DataSourcesPopupComponent,
    children: [
      {
        path: 'create-datasource',
        component: DataSourcesCreateDataSourcePopupComponent,
      },
    ],
  },
  {
    outlet: 'popup',
    path: 'container',
    component: ContainerPopupComponent,
    children: [
      {
        path: 'create-container',
        component: ContainerCreateContainerPopupComponent,
      },
    ],
  },
  {
    outlet: 'popup',
    path: 'projects',
    component: ProjectsPopupComponent,
    children: [
      {
        path: 'create-project',
        component: ProjectsCreateProjectPopupComponent,
      },
    ],
  },
  {
    outlet: 'popup',
    path: 'datastreams',
    component: DataStreamsPopupComponent,
    children: [
      {
        path: 'create-datastream',
        component: DataStreamsCreateDataStreamPopupComponent,
      },
    ],
  },
  {
    outlet: 'popup',
    path: 'edges',
    component: EdgesPopupComponent,
    children: [
      {
        path: 'create-edge',
        component: EdgesCreateEdgePopupComponent,
      },
    ],
  },
  {
    outlet: 'popup',
    path: 'welcome',
    component: WelcomePopupComponent,
    children: [
      {
        path: 'alpha',
        component: WelcomeAlphaPopupComponent,
      },
    ],
  },
];

@NgModule({
  imports: [
    RouterModule.forRoot(appRoutes, {
      enableTracing: true, // <-- debugging purposes only
    }),
    NgZorroAntdModule.forRoot(),
  ],
  exports: [RouterModule],
  providers: [{ provide: NZ_LOCALE, useValue: enUS }],
})
export class AppRoutingModule {}
