import { Component, OnInit, OnDestroy } from '@angular/core';
import { Router, ActivatedRoute } from '@angular/router';
import { Http } from '@angular/http';
import { RegistryService } from '../../services/registry.service';
import { getHttpRequestOptions } from '../../utils/httpUtil';
import { TableBaseComponent } from '../../base-components/table.base.component';

@Component({
  selector: 'app-category',
  templateUrl: './category.component.html',
  styleUrls: ['./category.component.css'],
})
export class CategoryComponent extends TableBaseComponent
  implements OnInit, OnDestroy {
  categoryId: string = null;
  categoryName: string = null;
  sub = null;
  category: any = {};
  categoryValues = [];

  constructor(
    router: Router,
    private http: Http,
    private route: ActivatedRoute,
    private registryService: RegistryService
  ) {
    super(router);
  }

  ngOnInit() {
    this.sub = this.route.params.subscribe(async params => {
      this.category = this.registryService.get(params['id']);
      this.categoryId = this.category.id;
      this.categoryValues = this.category.valuesInfo;
    });
  }

  ngOnDestroy() {
    this.sub.unsubscribe();
  }
}
