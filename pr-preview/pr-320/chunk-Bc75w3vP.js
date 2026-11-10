import{$ as ae,A as Mi,At as ra,Bt as xe$1,C as JD,Ct as nC,D as LG,Dt as pC,E as L,Et as no,F as Oe,Ft as ul,G as Ve,Gt as yv,H as U,Ht as yf,I as Ot,It as uv,J as Xi,K as Vo,Kt as zC,L as PC,Lt as v,M as O,Mt as sq,N as OI,Nt as tC,O as Le$1,Ot as pe,P as OM,Pt as uC,Q as _$,R as Pl,Rt as vf,S as J2,St as me,T as Ji,Tt as nn$1,U as Up,Ut as yo,V as Tn,Vt as ye$1,W as VN,Wt as yr,X as YE,Y as Xo,Z as Zn,_ as HE,_t as jt$1,a as BC,b as Io,bt as mC,c as D,ct as gv,d as EI,dt as he,et as bc,f as Ef,ft as hv,g as H,gt as jp,h as G,ht as jl,i as B,it as de,j as Mr,jt as rc,k as Lp,kt as qz,l as D$,lt as h8,m as Fi,mt as ji,n as z,nt as c8,o as Be,ot as g8,p as Et,pt as j,q as Wi,r as $C,rt as da,s as Bp,st as gc,t as Qe$1,tt as bo,u as Dv,ut as hc,v as Hi,vt as kf,w as Jd,wt as nb,x as J,xt as mG,y as If,yt as m,z as R,zt as xI}from"./main-PIB2L4BF.js";import{t as k}from"./chunk-jqROH7uh.js";var X=class{_multiple;_emitChanges;compareWith;_selection=/* @__PURE__ */ new Set;_deselectedToEmit=[];_selectedToEmit=[];_selected=null;get selected(){return this._selected||(this._selected=Array.from(this._selection.values())),this._selected}changed=new H;bulk={select:n=>this._select(n),deselect:n=>this._deselect(n),setSelection:n=>this._setSelection(n)};constructor(n=!1,t,e=!0,a){this._multiple=n,this._emitChanges=e,this.compareWith=a,t&&t.length&&(n?t.forEach(o=>this._markSelected(o)):this._markSelected(t[0]),this._selectedToEmit.length=0)}select(...n){return this._select(n)}deselect(...n){return this._deselect(n)}setSelection(...n){return this._setSelection(n)}toggle(n){return this.isSelected(n)?this.deselect(n):this.select(n)}clear(n=!0){this._unmarkAll();let t=this._hasQueuedChanges();return n&&this._emitChangeEvent(),t}isSelected(n){return this._selection.has(this._getConcreteValue(n))}isEmpty(){return this._selection.size===0}hasValue(){return!this.isEmpty()}sort(n){this._multiple&&this.selected&&this._selected.sort(n)}isMultipleSelection(){return this._multiple}_select(n){this._verifyValueAssignment(n),n.forEach(e=>this._markSelected(e));let t=this._hasQueuedChanges();return this._emitChangeEvent(),t}_deselect(n){this._verifyValueAssignment(n),n.forEach(e=>this._unmarkSelected(e));let t=this._hasQueuedChanges();return this._emitChangeEvent(),t}_setSelection(n){this._verifyValueAssignment(n);let t=this.selected,e=new Set(n.map(o=>this._getConcreteValue(o)));n.forEach(o=>this._markSelected(o)),t.filter(o=>!e.has(this._getConcreteValue(o,e))).forEach(o=>this._unmarkSelected(o));let a=this._hasQueuedChanges();return this._emitChangeEvent(),a}_emitChangeEvent(){this._selected=null,(this._selectedToEmit.length||this._deselectedToEmit.length)&&(this.changed.next({source:this,added:this._selectedToEmit,removed:this._deselectedToEmit}),this._deselectedToEmit=[],this._selectedToEmit=[])}_markSelected(n){n=this._getConcreteValue(n),this.isSelected(n)||(this._multiple||this._unmarkAll(),this.isSelected(n)||this._selection.add(n),this._emitChanges&&this._selectedToEmit.push(n))}_unmarkSelected(n){n=this._getConcreteValue(n),this.isSelected(n)&&(this._selection.delete(n),this._emitChanges&&this._deselectedToEmit.push(n))}_unmarkAll(){this.isEmpty()||this._selection.forEach(n=>this._unmarkSelected(n))}_verifyValueAssignment(n){n.length>1&&this._multiple}_hasQueuedChanges(){return!!(this._deselectedToEmit.length||this._selectedToEmit.length)}_getConcreteValue(n,t){if(this.compareWith){t=t??this._selection;for(let e of t)if(this.compareWith(n,e))return e;return n}else return n}};var ye=new v(``);var Ce=(()=>{class i{_animationsDisabled=Vo();state=`unchecked`;disabled=!1;appearance=`full`;static ɵfac=function(e){return new(e||i)};static ɵcmp=Ve({type:i,selectors:[[`mat-pseudo-checkbox`]],hostAttrs:[1,`mat-pseudo-checkbox`],hostVars:12,hostBindings:function(e,a){e&2&&Ot(`mat-pseudo-checkbox-indeterminate`,a.state===`indeterminate`)(`mat-pseudo-checkbox-checked`,a.state===`checked`)(`mat-pseudo-checkbox-disabled`,a.disabled)(`mat-pseudo-checkbox-minimal`,a.appearance===`minimal`)(`mat-pseudo-checkbox-full`,a.appearance===`full`)(`_mat-animation-noopable`,a._animationsDisabled)},inputs:{state:`state`,disabled:`disabled`,appearance:`appearance`},decls:0,vars:0,template:function(e,a){},styles:[`.mat-pseudo-checkbox {
  border-radius: 2px;
  cursor: pointer;
  display: inline-block;
  vertical-align: middle;
  box-sizing: border-box;
  position: relative;
  flex-shrink: 0;
  transition: border-color 90ms cubic-bezier(0, 0, 0.2, 0.1), background-color 90ms cubic-bezier(0, 0, 0.2, 0.1);
}
.mat-pseudo-checkbox::after {
  position: absolute;
  opacity: 0;
  content: "";
  border-bottom: 2px solid currentColor;
  transition: opacity 90ms cubic-bezier(0, 0, 0.2, 0.1);
}
.mat-pseudo-checkbox._mat-animation-noopable {
  transition: none !important;
  animation: none !important;
}
.mat-pseudo-checkbox._mat-animation-noopable::after {
  transition: none;
}

.mat-pseudo-checkbox-disabled {
  cursor: default;
}

.mat-pseudo-checkbox-indeterminate::after {
  left: 1px;
  opacity: 1;
  border-radius: 2px;
}

.mat-pseudo-checkbox-checked::after {
  left: 1px;
  border-left: 2px solid currentColor;
  transform: rotate(-45deg);
  opacity: 1;
  box-sizing: content-box;
}

.mat-pseudo-checkbox-minimal.mat-pseudo-checkbox-checked::after, .mat-pseudo-checkbox-minimal.mat-pseudo-checkbox-indeterminate::after {
  color: var(--%NS%mat-pseudo-checkbox-minimal-selected-checkmark-color, var(--%NS%mat-sys-primary));
}
.mat-pseudo-checkbox-minimal.mat-pseudo-checkbox-checked.mat-pseudo-checkbox-disabled::after, .mat-pseudo-checkbox-minimal.mat-pseudo-checkbox-indeterminate.mat-pseudo-checkbox-disabled::after {
  color: var(--%NS%mat-pseudo-checkbox-minimal-disabled-selected-checkmark-color, color-mix(in srgb, var(--%NS%mat-sys-on-surface) 38%, transparent));
}

.mat-pseudo-checkbox-full {
  border-color: var(--%NS%mat-pseudo-checkbox-full-unselected-icon-color, var(--%NS%mat-sys-on-surface-variant));
  border-width: 2px;
  border-style: solid;
}
.mat-pseudo-checkbox-full.mat-pseudo-checkbox-disabled {
  border-color: var(--%NS%mat-pseudo-checkbox-full-disabled-unselected-icon-color, color-mix(in srgb, var(--%NS%mat-sys-on-surface) 38%, transparent));
}
.mat-pseudo-checkbox-full.mat-pseudo-checkbox-checked, .mat-pseudo-checkbox-full.mat-pseudo-checkbox-indeterminate {
  background-color: var(--%NS%mat-pseudo-checkbox-full-selected-icon-color, var(--%NS%mat-sys-primary));
  border-color: transparent;
}
.mat-pseudo-checkbox-full.mat-pseudo-checkbox-checked::after, .mat-pseudo-checkbox-full.mat-pseudo-checkbox-indeterminate::after {
  color: var(--%NS%mat-pseudo-checkbox-full-selected-checkmark-color, var(--%NS%mat-sys-on-primary));
}
.mat-pseudo-checkbox-full.mat-pseudo-checkbox-checked.mat-pseudo-checkbox-disabled, .mat-pseudo-checkbox-full.mat-pseudo-checkbox-indeterminate.mat-pseudo-checkbox-disabled {
  background-color: var(--%NS%mat-pseudo-checkbox-full-disabled-selected-icon-color, color-mix(in srgb, var(--%NS%mat-sys-on-surface) 38%, transparent));
}
.mat-pseudo-checkbox-full.mat-pseudo-checkbox-checked.mat-pseudo-checkbox-disabled::after, .mat-pseudo-checkbox-full.mat-pseudo-checkbox-indeterminate.mat-pseudo-checkbox-disabled::after {
  color: var(--%NS%mat-pseudo-checkbox-full-disabled-selected-checkmark-color, var(--%NS%mat-sys-surface));
}

.mat-pseudo-checkbox {
  width: 18px;
  height: 18px;
}

.mat-pseudo-checkbox-minimal.mat-pseudo-checkbox-checked::after {
  width: 14px;
  height: 6px;
  transform-origin: center;
  top: -4.2426406871px;
  left: 0;
  bottom: 0;
  right: 0;
  margin: auto;
}
.mat-pseudo-checkbox-minimal.mat-pseudo-checkbox-indeterminate::after {
  top: 8px;
  width: 16px;
}

.mat-pseudo-checkbox-full.mat-pseudo-checkbox-checked::after {
  width: 10px;
  height: 4px;
  transform-origin: center;
  top: -2.8284271247px;
  left: 0;
  bottom: 0;
  right: 0;
  margin: auto;
}
.mat-pseudo-checkbox-full.mat-pseudo-checkbox-indeterminate::after {
  top: 6px;
  width: 12px;
}
`],encapsulation:2})}return i})();var xe=new v(`MAT_BUTTON_TOGGLE_DEFAULT_OPTIONS`,{providedIn:`root`,factory:()=>({hideSingleSelectionIndicator:!1,hideMultipleSelectionIndicator:!1,disabledInteractive:!1})});var ke=new v(`MatButtonToggleGroup`);var Le={provide:ye,useExisting:da(()=>Ft),multi:!0};var _t=class{source;value;constructor(n,t){this.source=n,this.value=t}};var Ft=(()=>{class i{_changeDetector=m(Wi);_dir=m(VN,{optional:!0});_multiple=!1;_disabled=!1;_disabledInteractive=!1;_selectionModel;_rawValue;_controlValueAccessorChangeFn=()=>{};_onTouched=()=>{};_buttonToggles;appearance;get name(){return this._name}set name(t){this._name=t,this._markButtonsForCheck()}_name=m(Up).getId(`mat-button-toggle-group-`);vertical=!1;get value(){let t=this._selectionModel?this._selectionModel.selected:[];return this.multiple?t.map(e=>e.value):t[0]?t[0].value:void 0}set value(t){this._setSelectionByValue(t),this.valueChange.emit(this.value)}valueChange=new ae;get selected(){let t=this._selectionModel?this._selectionModel.selected:[];return this.multiple?t:t[0]||null}get multiple(){return this._multiple}set multiple(t){this._multiple=t,this._markButtonsForCheck()}get disabled(){return this._disabled}set disabled(t){this._disabled=t,this._markButtonsForCheck()}get disabledInteractive(){return this._disabledInteractive}set disabledInteractive(t){this._disabledInteractive=t,this._markButtonsForCheck()}get dir(){return this._dir&&this._dir.value===`rtl`?`rtl`:`ltr`}change=new ae;get hideSingleSelectionIndicator(){return this._hideSingleSelectionIndicator}set hideSingleSelectionIndicator(t){this._hideSingleSelectionIndicator=t,this._markButtonsForCheck()}_hideSingleSelectionIndicator;get hideMultipleSelectionIndicator(){return this._hideMultipleSelectionIndicator}set hideMultipleSelectionIndicator(t){this._hideMultipleSelectionIndicator=t,this._markButtonsForCheck()}_hideMultipleSelectionIndicator;constructor(){let t=m(xe,{optional:!0});this.appearance=t&&t.appearance?t.appearance:`standard`,this._hideSingleSelectionIndicator=t?.hideSingleSelectionIndicator??!1,this._hideMultipleSelectionIndicator=t?.hideMultipleSelectionIndicator??!1}ngOnInit(){this._selectionModel=new X(this.multiple,void 0,!1)}ngAfterContentInit(){this._selectionModel.select(...this._buttonToggles.filter(t=>t.checked)),this.multiple||this._initializeTabIndex()}writeValue(t){this.value=t,this._changeDetector.markForCheck()}registerOnChange(t){this._controlValueAccessorChangeFn=t}registerOnTouched(t){this._onTouched=t}setDisabledState(t){this.disabled=t}_keydown(t){if(this.multiple||this.disabled||HE(t))return;let a=t.target.id,o=this._buttonToggles.toArray().findIndex(r=>r.buttonId===a),s=null;switch(t.keyCode){case 32:case 13:s=this._buttonToggles.get(o)||null;break;case 38:s=this._getNextButton(o,-1);break;case 37:s=this._getNextButton(o,this.dir===`ltr`?-1:1);break;case 40:s=this._getNextButton(o,1);break;case 39:s=this._getNextButton(o,this.dir===`ltr`?1:-1);break;default:return}s&&(t.preventDefault(),s._onButtonClick(),s.focus())}_emitChangeEvent(t){let e=new _t(t,this.value);this._rawValue=e.value,this._controlValueAccessorChangeFn(e.value),this.change.emit(e)}_syncButtonToggle(t,e,a=!1,o=!1){!this.multiple&&this.selected&&!t.checked&&(this.selected.checked=!1),this._selectionModel?e?this._selectionModel.select(t):this._selectionModel.deselect(t):o=!0,o?Promise.resolve().then(()=>this._updateModelValue(t,a)):this._updateModelValue(t,a)}_isSelected(t){return this._selectionModel&&this._selectionModel.isSelected(t)}_isPrechecked(t){return typeof this._rawValue>`u`?!1:this.multiple&&Array.isArray(this._rawValue)?this._rawValue.some(e=>t.value!=null&&e===t.value):t.value===this._rawValue}_initializeTabIndex(){if(this._buttonToggles.forEach(t=>{t.tabIndex=-1}),this.selected)this.selected.tabIndex=0;else for(let t=0;t<this._buttonToggles.length;t++){let e=this._buttonToggles.get(t);if(!e.disabled){e.tabIndex=0;break}}}_getNextButton(t,e){let a=this._buttonToggles;for(let o=1;o<=a.length;o++){let s=(t+e*o+a.length)%a.length,r=a.get(s);if(r&&!r.disabled)return r}return null}_setSelectionByValue(t){if(this._rawValue=t,!this._buttonToggles)return;let e=this._buttonToggles.toArray();if(this.multiple&&t?(this._clearSelection(),t.forEach(a=>this._selectValue(a,e))):(this._clearSelection(),this._selectValue(t,e)),!this.multiple&&e.every(a=>a.tabIndex===-1)){for(let a of e)if(!a.disabled){a.tabIndex=0;break}}}_clearSelection(){this._selectionModel.clear(),this._buttonToggles.forEach(t=>{t.checked=!1,this.multiple||(t.tabIndex=-1)})}_selectValue(t,e){for(let a of e)if(a.value===t){a.checked=!0,this._selectionModel.select(a),this.multiple||(a.tabIndex=0);break}}_updateModelValue(t,e){e&&this._emitChangeEvent(t),this.valueChange.emit(this.value)}_markButtonsForCheck(){this._buttonToggles?.forEach(t=>t._markForCheck())}static ɵfac=function(e){return new(e||i)};static ɵdir=Be({type:i,selectors:[[`mat-button-toggle-group`]],contentQueries:function(e,a,o){if(e&1&&bc(o,Pt,5),e&2){let s;yf(s=Ef())&&(a._buttonToggles=s)}},hostAttrs:[1,`mat-button-toggle-group`],hostVars:6,hostBindings:function(e,a){e&1&&gc(`keydown`,function(s){return a._keydown(s)}),e&2&&(Tn(`role`,a.multiple?`group`:`radiogroup`)(`aria-disabled`,a.disabled),Ot(`mat-button-toggle-vertical`,a.vertical)(`mat-button-toggle-group-appearance-standard`,a.appearance===`standard`))},inputs:{appearance:`appearance`,name:`name`,vertical:[2,`vertical`,`vertical`,xe$1],value:`value`,multiple:[2,`multiple`,`multiple`,xe$1],disabled:[2,`disabled`,`disabled`,xe$1],disabledInteractive:[2,`disabledInteractive`,`disabledInteractive`,xe$1],hideSingleSelectionIndicator:[2,`hideSingleSelectionIndicator`,`hideSingleSelectionIndicator`,xe$1],hideMultipleSelectionIndicator:[2,`hideMultipleSelectionIndicator`,`hideMultipleSelectionIndicator`,xe$1]},outputs:{valueChange:`valueChange`,change:`change`},exportAs:[`matButtonToggleGroup`],features:[BC([Le,{provide:ke,useExisting:i}])]})}return i})();var Pt=(()=>{class i{_changeDetectorRef=m(Wi);_elementRef=m(de);_focusMonitor=m(Lp);_idGenerator=m(Up);_animationDisabled=Vo();_checked=!1;ariaLabel;ariaLabelledby=null;_buttonElement;buttonToggleGroup;get buttonId(){return`${this.id}-button`}id;name;value;get tabIndex(){return this._tabIndex()}set tabIndex(t){this._tabIndex.set(t)}_tabIndex;disableRipple=!1;get appearance(){return this.buttonToggleGroup?this.buttonToggleGroup.appearance:this._appearance}set appearance(t){this._appearance=t}_appearance;get checked(){return this.buttonToggleGroup?this.buttonToggleGroup._isSelected(this):this._checked}set checked(t){t!==this._checked&&(this._checked=t,this.buttonToggleGroup&&this.buttonToggleGroup._syncButtonToggle(this,this._checked),this._changeDetectorRef.markForCheck())}get disabled(){return this._disabled||this.buttonToggleGroup&&this.buttonToggleGroup.disabled}set disabled(t){this._disabled=t}_disabled=!1;get disabledInteractive(){return this._disabledInteractive||this.buttonToggleGroup!==null&&this.buttonToggleGroup.disabledInteractive}set disabledInteractive(t){this._disabledInteractive=t}_disabledInteractive;change=new ae;constructor(){m(Ji).load(YE);let t=m(ke,{optional:!0}),e=m(new yo(`tabindex`),{optional:!0})||``,a=m(xe,{optional:!0});this._tabIndex=B(parseInt(e)||0),this.buttonToggleGroup=t,this._appearance=a&&a.appearance?a.appearance:`standard`,this._disabledInteractive=a?.disabledInteractive??!1}ngOnInit(){let t=this.buttonToggleGroup;this.id=this.id||this._idGenerator.getId(`mat-button-toggle-`),t&&(t._isPrechecked(this)?this.checked=!0:t._isSelected(this)!==this._checked&&t._syncButtonToggle(this,this._checked))}ngAfterViewInit(){this._animationDisabled||this._elementRef.nativeElement.classList.add(`mat-button-toggle-animations-enabled`),this._focusMonitor.monitor(this._elementRef,!0)}ngOnDestroy(){let t=this.buttonToggleGroup;this._focusMonitor.stopMonitoring(this._elementRef),t&&t._isSelected(this)&&t._syncButtonToggle(this,!1,!1,!0)}focus(t){this._buttonElement.nativeElement.focus(t)}_onButtonClick(){if(this.disabled)return;let t=this.isSingleSelector()?!0:!this._checked;if(t!==this._checked&&(this._checked=t,this.buttonToggleGroup&&(this.buttonToggleGroup._syncButtonToggle(this,this._checked,!0),this.buttonToggleGroup._onTouched())),this.isSingleSelector()){let e=this.buttonToggleGroup._buttonToggles.find(a=>a.tabIndex===0);e&&(e.tabIndex=-1),this.tabIndex=0}this.change.emit(new _t(this,this.value))}_markForCheck(){this._changeDetectorRef.markForCheck()}_getButtonName(){return this.isSingleSelector()?this.buttonToggleGroup.name:this.name||null}isSingleSelector(){return this.buttonToggleGroup&&!this.buttonToggleGroup.multiple}static ɵfac=function(e){return new(e||i)};static ɵcmp=(function(){let t=[`button`],e=[`*`];function a(o,s){if(o&1&&(rc(0,`div`,2),hc(1,`mat-pseudo-checkbox`,6),vf()),o&2){let r=uC();Mi(),uv(`disabled`,r.disabled)}}return Ve({type:i,selectors:[[`mat-button-toggle`]],viewQuery:function(s,r){if(s&1&&hv(t,5),s&2){let f;yf(f=Ef())&&(r._buttonElement=f.first)}},hostAttrs:[`role`,`presentation`,1,`mat-button-toggle`],hostVars:14,hostBindings:function(s,r){s&1&&gc(`focus`,function(){return r.focus()}),s&2&&(Tn(`aria-label`,null)(`aria-labelledby`,null)(`id`,r.id)(`name`,null),Ot(`mat-button-toggle-standalone`,!r.buttonToggleGroup)(`mat-button-toggle-checked`,r.checked)(`mat-button-toggle-disabled`,r.disabled)(`mat-button-toggle-disabled-interactive`,r.disabledInteractive)(`mat-button-toggle-appearance-standard`,r.appearance===`standard`))},inputs:{ariaLabel:[0,`aria-label`,`ariaLabel`],ariaLabelledby:[0,`aria-labelledby`,`ariaLabelledby`],id:`id`,name:`name`,value:`value`,tabIndex:`tabIndex`,disableRipple:[2,`disableRipple`,`disableRipple`,xe$1],appearance:`appearance`,checked:[2,`checked`,`checked`,xe$1],disabled:[2,`disabled`,`disabled`,xe$1],disabledInteractive:[2,`disabledInteractive`,`disabledInteractive`,xe$1]},outputs:{change:`change`},exportAs:[`matButtonToggle`],ngContentSelectors:e,decls:7,vars:13,consts:[[`button`,``],[`type`,`button`,1,`mat-button-toggle-button`,`mat-focus-indicator`,3,`click`,`id`,`disabled`],[1,`mat-button-toggle-checkbox-wrapper`],[1,`mat-button-toggle-label-content`],[1,`mat-button-toggle-focus-overlay`],[`matRipple`,``,1,`mat-button-toggle-ripple`,3,`matRippleTrigger`,`matRippleDisabled`],[`state`,`checked`,`aria-hidden`,`true`,`appearance`,`minimal`,3,`disabled`]],template:function(s,r){if(s&1&&(yr(),rc(0,`button`,1,0),gc(`click`,function(){return r._onButtonClick()}),Fi(2,a,2,1,`div`,2),rc(3,`span`,3),Et(4),vf()(),hc(5,`span`,4)(6,`span`,5)),s&2){let f=mC(1);uv(`id`,r.buttonId)(`disabled`,r.disabled&&!r.disabledInteractive||null),Tn(`role`,r.isSingleSelector()?`radio`:`button`)(`tabindex`,r.disabled&&!r.disabledInteractive?-1:r.tabIndex)(`aria-pressed`,r.isSingleSelector()?null:r.checked)(`aria-checked`,r.isSingleSelector()?r.checked:null)(`name`,r._getButtonName())(`aria-label`,r.ariaLabel)(`aria-labelledby`,r.ariaLabelledby)(`aria-disabled`,r.disabled&&r.disabledInteractive?`true`:null),Mi(2),ji(r.buttonToggleGroup&&(!r.buttonToggleGroup.multiple&&!r.buttonToggleGroup.hideSingleSelectionIndicator||r.buttonToggleGroup.multiple&&!r.buttonToggleGroup.hideMultipleSelectionIndicator)?2:-1),Mi(4),uv(`matRippleTrigger`,f)(`matRippleDisabled`,r.disableRipple||r.disabled)}},dependencies:[mG,Ce],styles:[`.mat-button-toggle-standalone,
.mat-button-toggle-group {
  position: relative;
  display: inline-flex;
  flex-direction: row;
  white-space: nowrap;
  overflow: hidden;
  -webkit-tap-highlight-color: transparent;
  border-radius: var(--%NS%mat-button-toggle-legacy-shape);
  transform: translateZ(0);
}
.mat-button-toggle-standalone:not([class*=mat-elevation-z]),
.mat-button-toggle-group:not([class*=mat-elevation-z]) {
  box-shadow: 0px 3px 1px -2px rgba(0, 0, 0, 0.2), 0px 2px 2px 0px rgba(0, 0, 0, 0.14), 0px 1px 5px 0px rgba(0, 0, 0, 0.12);
}
@media (forced-colors: active) {
  .mat-button-toggle-standalone,
  .mat-button-toggle-group {
    outline: solid 1px;
  }
}

.mat-button-toggle-standalone.mat-button-toggle-appearance-standard,
.mat-button-toggle-group-appearance-standard {
  border-radius: var(--%NS%mat-button-toggle-shape, var(--%NS%mat-sys-corner-extra-large));
  border: solid 1px var(--%NS%mat-button-toggle-divider-color, var(--%NS%mat-sys-outline));
}
.mat-button-toggle-standalone.mat-button-toggle-appearance-standard .mat-pseudo-checkbox,
.mat-button-toggle-group-appearance-standard .mat-pseudo-checkbox {
  --%NS%mat-pseudo-checkbox-minimal-selected-checkmark-color: var(--%NS%mat-button-toggle-selected-state-text-color, var(--%NS%mat-sys-on-secondary-container));
}
.mat-button-toggle-standalone.mat-button-toggle-appearance-standard:not([class*=mat-elevation-z]),
.mat-button-toggle-group-appearance-standard:not([class*=mat-elevation-z]) {
  box-shadow: none;
}
@media (forced-colors: active) {
  .mat-button-toggle-standalone.mat-button-toggle-appearance-standard,
  .mat-button-toggle-group-appearance-standard {
    outline: 0;
  }
}

.mat-button-toggle-vertical {
  flex-direction: column;
}
.mat-button-toggle-vertical .mat-button-toggle-label-content {
  display: block;
}

.mat-button-toggle {
  white-space: nowrap;
  position: relative;
  color: var(--%NS%mat-button-toggle-legacy-text-color);
  font-family: var(--%NS%mat-button-toggle-legacy-label-text-font);
  font-size: var(--%NS%mat-button-toggle-legacy-label-text-size);
  line-height: var(--%NS%mat-button-toggle-legacy-label-text-line-height);
  font-weight: var(--%NS%mat-button-toggle-legacy-label-text-weight);
  letter-spacing: var(--%NS%mat-button-toggle-legacy-label-text-tracking);
  --%NS%mat-pseudo-checkbox-minimal-selected-checkmark-color: var(--%NS%mat-button-toggle-legacy-selected-state-text-color);
}
.mat-button-toggle.cdk-keyboard-focused .mat-button-toggle-focus-overlay {
  opacity: var(--%NS%mat-button-toggle-legacy-focus-state-layer-opacity);
}
.mat-button-toggle .mat-icon svg {
  vertical-align: top;
}

.mat-button-toggle-checkbox-wrapper {
  display: inline-block;
  justify-content: flex-start;
  align-items: center;
  width: 0;
  height: 18px;
  line-height: 18px;
  overflow: hidden;
  box-sizing: border-box;
  position: absolute;
  top: 50%;
  left: 16px;
  transform: translate3d(0, -50%, 0);
}
[dir=rtl] .mat-button-toggle-checkbox-wrapper {
  left: auto;
  right: 16px;
}
.mat-button-toggle-appearance-standard .mat-button-toggle-checkbox-wrapper {
  left: 12px;
}
[dir=rtl] .mat-button-toggle-appearance-standard .mat-button-toggle-checkbox-wrapper {
  left: auto;
  right: 12px;
}
.mat-button-toggle-checked .mat-button-toggle-checkbox-wrapper {
  width: 18px;
}
.mat-button-toggle-animations-enabled .mat-button-toggle-checkbox-wrapper {
  transition: width 150ms 45ms cubic-bezier(0.4, 0, 0.2, 1);
}
.mat-button-toggle-vertical .mat-button-toggle-checkbox-wrapper {
  transition: none;
}

.mat-button-toggle-checked {
  color: var(--%NS%mat-button-toggle-legacy-selected-state-text-color);
  background-color: var(--%NS%mat-button-toggle-legacy-selected-state-background-color);
}

.mat-button-toggle-disabled {
  pointer-events: none;
  color: var(--%NS%mat-button-toggle-legacy-disabled-state-text-color);
  background-color: var(--%NS%mat-button-toggle-legacy-disabled-state-background-color);
  --%NS%mat-pseudo-checkbox-minimal-disabled-selected-checkmark-color: var(--%NS%mat-button-toggle-legacy-disabled-state-text-color);
}
.mat-button-toggle-disabled.mat-button-toggle-checked {
  background-color: var(--%NS%mat-button-toggle-legacy-disabled-selected-state-background-color);
}

.mat-button-toggle-disabled-interactive {
  pointer-events: auto;
}

.mat-button-toggle-appearance-standard {
  color: var(--%NS%mat-button-toggle-text-color, var(--%NS%mat-sys-on-surface));
  background-color: var(--%NS%mat-button-toggle-background-color, transparent);
  font-family: var(--%NS%mat-button-toggle-label-text-font, var(--%NS%mat-sys-label-large-font));
  font-size: var(--%NS%mat-button-toggle-label-text-size, var(--%NS%mat-sys-label-large-size));
  line-height: var(--%NS%mat-button-toggle-label-text-line-height, var(--%NS%mat-sys-label-large-line-height));
  font-weight: var(--%NS%mat-button-toggle-label-text-weight, var(--%NS%mat-sys-label-large-weight));
  letter-spacing: var(--%NS%mat-button-toggle-label-text-tracking, var(--%NS%mat-sys-label-large-tracking));
}
.mat-button-toggle-group-appearance-standard .mat-button-toggle-appearance-standard + .mat-button-toggle-appearance-standard {
  border-left: solid 1px var(--%NS%mat-button-toggle-divider-color, var(--%NS%mat-sys-outline));
}
[dir=rtl] .mat-button-toggle-group-appearance-standard .mat-button-toggle-appearance-standard + .mat-button-toggle-appearance-standard {
  border-left: none;
  border-right: solid 1px var(--%NS%mat-button-toggle-divider-color, var(--%NS%mat-sys-outline));
}
.mat-button-toggle-group-appearance-standard.mat-button-toggle-vertical .mat-button-toggle-appearance-standard + .mat-button-toggle-appearance-standard {
  border-left: none;
  border-right: none;
  border-top: solid 1px var(--%NS%mat-button-toggle-divider-color, var(--%NS%mat-sys-outline));
}
.mat-button-toggle-appearance-standard.mat-button-toggle-checked {
  color: var(--%NS%mat-button-toggle-selected-state-text-color, var(--%NS%mat-sys-on-secondary-container));
  background-color: var(--%NS%mat-button-toggle-selected-state-background-color, var(--%NS%mat-sys-secondary-container));
}
.mat-button-toggle-appearance-standard.mat-button-toggle-disabled {
  color: var(--%NS%mat-button-toggle-disabled-state-text-color, color-mix(in srgb, var(--%NS%mat-sys-on-surface) 38%, transparent));
  background-color: var(--%NS%mat-button-toggle-disabled-state-background-color, transparent);
}
.mat-button-toggle-appearance-standard.mat-button-toggle-disabled .mat-pseudo-checkbox {
  --%NS%mat-pseudo-checkbox-minimal-disabled-selected-checkmark-color: var(--%NS%mat-button-toggle-disabled-selected-state-text-color, color-mix(in srgb, var(--%NS%mat-sys-on-surface) 38%, transparent));
}
.mat-button-toggle-appearance-standard.mat-button-toggle-disabled.mat-button-toggle-checked {
  color: var(--%NS%mat-button-toggle-disabled-selected-state-text-color, color-mix(in srgb, var(--%NS%mat-sys-on-surface) 38%, transparent));
  background-color: var(--%NS%mat-button-toggle-disabled-selected-state-background-color, color-mix(in srgb, var(--%NS%mat-sys-on-surface) 12%, transparent));
}
.mat-button-toggle-appearance-standard .mat-button-toggle-focus-overlay {
  background-color: var(--%NS%mat-button-toggle-state-layer-color, var(--%NS%mat-sys-on-surface));
}
.mat-button-toggle-appearance-standard:hover .mat-button-toggle-focus-overlay {
  opacity: var(--%NS%mat-button-toggle-hover-state-layer-opacity, var(--%NS%mat-sys-hover-state-layer-opacity));
}
.mat-button-toggle-appearance-standard.cdk-keyboard-focused .mat-button-toggle-focus-overlay {
  opacity: var(--%NS%mat-button-toggle-focus-state-layer-opacity, var(--%NS%mat-sys-focus-state-layer-opacity));
}
@media (hover: none) {
  .mat-button-toggle-appearance-standard:hover .mat-button-toggle-focus-overlay {
    display: none;
  }
}

.mat-button-toggle-label-content {
  -webkit-user-select: none;
  user-select: none;
  display: inline-block;
  padding: 0 16px;
  line-height: var(--%NS%mat-button-toggle-legacy-height);
  position: relative;
}
.mat-button-toggle-appearance-standard .mat-button-toggle-label-content {
  padding: 0 12px;
  line-height: var(--%NS%mat-button-toggle-height, 40px);
}

.mat-button-toggle-label-content > * {
  vertical-align: middle;
}

.mat-button-toggle-focus-overlay {
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  position: absolute;
  border-radius: inherit;
  pointer-events: none;
  opacity: 0;
  background-color: var(--%NS%mat-button-toggle-legacy-state-layer-color);
}

@media (forced-colors: active) {
  .mat-button-toggle-checked .mat-button-toggle-focus-overlay {
    border-bottom: solid 500px;
    opacity: 0.5;
    height: 0;
  }
  .mat-button-toggle-checked:hover .mat-button-toggle-focus-overlay {
    opacity: 0.6;
  }
  .mat-button-toggle-checked.mat-button-toggle-appearance-standard .mat-button-toggle-focus-overlay {
    border-bottom: solid 500px;
  }
}
.mat-button-toggle .mat-button-toggle-ripple {
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  position: absolute;
  pointer-events: none;
}

.mat-button-toggle-button {
  border: 0;
  background: none;
  color: inherit;
  padding: 0;
  margin: 0;
  font: inherit;
  outline: none;
  width: 100%;
  cursor: pointer;
}
.mat-button-toggle-animations-enabled .mat-button-toggle-button {
  transition: padding 150ms 45ms cubic-bezier(0.4, 0, 0.2, 1);
}
.mat-button-toggle-vertical .mat-button-toggle-button {
  transition: none;
}
.mat-button-toggle-disabled .mat-button-toggle-button {
  cursor: default;
}
.mat-button-toggle-button::-moz-focus-inner {
  border: 0;
}
.mat-button-toggle-checked .mat-button-toggle-button:has(.mat-button-toggle-checkbox-wrapper) {
  padding-left: 30px;
}
[dir=rtl] .mat-button-toggle-checked .mat-button-toggle-button:has(.mat-button-toggle-checkbox-wrapper) {
  padding-left: 0;
  padding-right: 30px;
}

.mat-button-toggle-standalone.mat-button-toggle-appearance-standard {
  --%NS%mat-focus-indicator-border-radius: var(--%NS%mat-button-toggle-shape, var(--%NS%mat-sys-corner-extra-large));
}

.mat-button-toggle-group-appearance-standard:not(.mat-button-toggle-vertical) .mat-button-toggle:last-of-type .mat-button-toggle-button::before {
  border-top-right-radius: var(--%NS%mat-button-toggle-shape, var(--%NS%mat-sys-corner-extra-large));
  border-bottom-right-radius: var(--%NS%mat-button-toggle-shape, var(--%NS%mat-sys-corner-extra-large));
}
.mat-button-toggle-group-appearance-standard:not(.mat-button-toggle-vertical) .mat-button-toggle:first-of-type .mat-button-toggle-button::before {
  border-top-left-radius: var(--%NS%mat-button-toggle-shape, var(--%NS%mat-sys-corner-extra-large));
  border-bottom-left-radius: var(--%NS%mat-button-toggle-shape, var(--%NS%mat-sys-corner-extra-large));
}

.mat-button-toggle-group-appearance-standard.mat-button-toggle-vertical .mat-button-toggle:last-of-type .mat-button-toggle-button::before {
  border-bottom-right-radius: var(--%NS%mat-button-toggle-shape, var(--%NS%mat-sys-corner-extra-large));
  border-bottom-left-radius: var(--%NS%mat-button-toggle-shape, var(--%NS%mat-sys-corner-extra-large));
}
.mat-button-toggle-group-appearance-standard.mat-button-toggle-vertical .mat-button-toggle:first-of-type .mat-button-toggle-button::before {
  border-top-right-radius: var(--%NS%mat-button-toggle-shape, var(--%NS%mat-sys-corner-extra-large));
  border-top-left-radius: var(--%NS%mat-button-toggle-shape, var(--%NS%mat-sys-corner-extra-large));
}
`],encapsulation:2})})()}return i})();var Bt=class{_box;_destroyed=new H;_resizeSubject=new H;_resizeObserver;_elementObservables=/* @__PURE__ */ new Map;constructor(n){this._box=n,typeof ResizeObserver<`u`&&(this._resizeObserver=new ResizeObserver(t=>this._resizeSubject.next(t)))}observe(n){return this._elementObservables.has(n)||this._elementObservables.set(n,new R(t=>{let e=this._resizeSubject.subscribe(t);return this._resizeObserver?.observe(n,{box:this._box}),()=>{this._resizeObserver?.unobserve(n),e.unsubscribe(),this._elementObservables.delete(n)}}).pipe(j(t=>t.filter(e=>e.target===n)),he(t=>t.length>0),OI({bufferSize:1,refCount:!0}),Zn(this._destroyed))),this._elementObservables.get(n)}destroy(){this._destroyed.next(),this._destroyed.complete(),this._resizeSubject.complete(),this._elementObservables.clear()}};var Se=(()=>{class i{_cleanupErrorListener;_observers=/* @__PURE__ */ new Map;_ngZone=m(G);constructor(){}ngOnDestroy(){for(let[,t]of this._observers)t.destroy();this._observers.clear(),this._cleanupErrorListener?.()}observe(t,e){let a=e?.box||`content-box`;return this._observers.has(a)||this._observers.set(a,new Bt(a)),this._observers.get(a).observe(t)}static ɵfac=function(e){return new(e||i)};static ɵprov=O({token:i,factory:i.ɵfac})}return i})();var Lt=`mdc-tab-indicator--active`;var Me=`mdc-tab-indicator--no-transition`;var Gt=class{_items;_currentItem;constructor(n){this._items=n}hide(){this._items.forEach(n=>n.deactivateInkBar()),this._currentItem=void 0}alignToElement(n){let t=this._items.find(a=>a.elementRef.nativeElement===n),e=this._currentItem;if(t!==e&&(e?.deactivateInkBar(),t)){let a=e?.elementRef.nativeElement.getBoundingClientRect?.();t.activateInkBar(a),this._currentItem=t}}};var Ge=(()=>{class i{_elementRef=m(de);_inkBarElement=null;_inkBarContentElement=null;_fitToContent=!1;get fitInkBarToContent(){return this._fitToContent}set fitInkBarToContent(t){this._fitToContent!==t&&(this._fitToContent=t,this._inkBarElement&&this._appendInkBarElement())}activateInkBar(t){let e=this._elementRef.nativeElement;if(!t||!e.getBoundingClientRect||!this._inkBarContentElement){e.classList.add(Lt);return}let a=e.getBoundingClientRect(),o=t.width/a.width,s=t.left-a.left;e.classList.add(Me),this._inkBarContentElement.style.setProperty(`transform`,`translateX(${s}px) scaleX(${o})`),e.getBoundingClientRect(),e.classList.remove(Me),e.classList.add(Lt),this._inkBarContentElement.style.setProperty(`transform`,``)}deactivateInkBar(){this._elementRef.nativeElement.classList.remove(Lt)}ngOnInit(){this._createInkBarElement()}ngOnDestroy(){this._inkBarElement?.remove(),this._inkBarElement=this._inkBarContentElement=null}_createInkBarElement(){let t=this._elementRef.nativeElement.ownerDocument||document,e=this._inkBarElement=t.createElement(`span`),a=this._inkBarContentElement=t.createElement(`span`);e.className=`mdc-tab-indicator`,a.className=`mdc-tab-indicator__content mdc-tab-indicator__content--underline`,e.appendChild(this._inkBarContentElement),this._appendInkBarElement()}_appendInkBarElement(){this._inkBarElement;(this._fitToContent?this._elementRef.nativeElement.querySelector(`.mdc-tab__content`):this._elementRef.nativeElement).appendChild(this._inkBarElement)}static ɵfac=function(e){return new(e||i)};static ɵdir=Be({type:i,inputs:{fitInkBarToContent:[2,`fitInkBarToContent`,`fitInkBarToContent`,xe$1]}})}return i})();var De={passive:!0};var je=650;var He=100;function ze(i){let n=i+``;return/^[0-9]+(?:\.[0-9]+)?$/.test(n)?`${i}ms`:/^[0-9]+(?:\.[0-9]+)?(?:ms|s)$/.test(n)?n:``}var $e=(()=>{class i{_elementRef=m(de);_changeDetectorRef=m(Wi);_viewportRuler=m(Qe$1);_dir=m(VN,{optional:!0});_ngZone=m(G);_platform=m(jt$1);_sharedResizeObserver=m(Se);_injector=m(J);_renderer=m(nn$1);_animationsDisabled=Vo();_eventCleanups;_scrollDistance=0;_selectedIndexChanged=!1;_destroyed=new H;_showPaginationControls=!1;_disableScrollAfter=!0;_disableScrollBefore=!0;_tabLabelCount;_scrollDistanceChanged=!1;_keyManager;_currentTextContent;_stopScrolling=new H;disablePagination=!1;get selectedIndex(){return this._selectedIndex}set selectedIndex(t){let e=isNaN(t)?0:t;this._selectedIndex!=e&&(this._selectedIndexChanged=!0,this._selectedIndex=e,this._keyManager&&this._keyManager.updateActiveItem(e))}_selectedIndex=0;selectFocusedIndex=new ae;indexFocused=new ae;constructor(){this._eventCleanups=this._ngZone.runOutsideAngular(()=>[this._renderer.listen(this._elementRef.nativeElement,`mouseleave`,()=>this._stopInterval())])}ngAfterViewInit(){this._eventCleanups.push(this._renderer.listen(this._previousPaginator.nativeElement,`touchstart`,()=>this._handlePaginatorPress(`before`),De),this._renderer.listen(this._nextPaginator.nativeElement,`touchstart`,()=>this._handlePaginatorPress(`after`),De))}ngAfterContentInit(){let t=this._dir?this._dir.change:D(`ltr`),e=this._sharedResizeObserver.observe(this._elementRef.nativeElement).pipe(Xo(32),Zn(this._destroyed)),a=this._viewportRuler.change(150).pipe(Zn(this._destroyed)),o=()=>{this.updatePagination(),this._alignInkBarToSelectedTab()};this._keyManager=new jp(this._items).withHorizontalOrientation(this._getLayoutDirection()).withHomeAndEnd().withWrap().skipPredicate(()=>!1),this._keyManager.updateActiveItem(Math.max(this._selectedIndex,0)),Jd(o,{injector:this._injector}),xI(t,a,e,this._items.changes,this._itemsResized()).pipe(Zn(this._destroyed)).subscribe(()=>{this._ngZone.run(()=>{Promise.resolve().then(()=>{this._scrollDistance=Math.max(0,Math.min(this._getMaxScrollDistance(),this._scrollDistance)),o()})}),this._keyManager?.withHorizontalOrientation(this._getLayoutDirection())}),this._keyManager.change.subscribe(s=>{this.indexFocused.emit(s),this._setTabFocus(s)})}_itemsResized(){return typeof ResizeObserver!=`function`?me:this._items.changes.pipe(jl(this._items),Oe(t=>new R(e=>this._ngZone.runOutsideAngular(()=>{let a=new ResizeObserver(o=>e.next(o));return t.forEach(o=>a.observe(o.elementRef.nativeElement)),()=>{a.disconnect()}}))),ra(1),he(t=>t.some(e=>e.contentRect.width>0&&e.contentRect.height>0)))}ngAfterContentChecked(){this._tabLabelCount!=this._items.length&&(this.updatePagination(),this._tabLabelCount=this._items.length,this._changeDetectorRef.markForCheck()),this._selectedIndexChanged&&(this._scrollToLabel(this._selectedIndex),this._checkScrollingControls(),this._alignInkBarToSelectedTab(),this._selectedIndexChanged=!1,this._changeDetectorRef.markForCheck()),this._scrollDistanceChanged&&(this._updateTabScrollPosition(),this._scrollDistanceChanged=!1,this._changeDetectorRef.markForCheck())}ngOnDestroy(){this._eventCleanups.forEach(t=>t()),this._keyManager?.destroy(),this._destroyed.next(),this._destroyed.complete(),this._stopScrolling.complete()}_handleKeydown(t){if(!HE(t))switch(t.keyCode){case 13:case 32:if(this.focusIndex!==this.selectedIndex){let e=this._items.get(this.focusIndex);e&&!e.disabled&&(this.selectFocusedIndex.emit(this.focusIndex),this._itemSelected(t))}break;default:this._keyManager?.onKeydown(t)}}_onContentChanges(){let t=this._elementRef.nativeElement.textContent;t!==this._currentTextContent&&(this._currentTextContent=t||``,this._ngZone.run(()=>{this.updatePagination(),this._alignInkBarToSelectedTab(),this._changeDetectorRef.markForCheck()}))}updatePagination(){this._checkPaginationEnabled(),this._checkScrollingControls(),this._updateTabScrollPosition()}get focusIndex(){return this._keyManager?this._keyManager.activeItemIndex:0}set focusIndex(t){!this._isValidIndex(t)||this.focusIndex===t||!this._keyManager||this._keyManager.setActiveItem(t)}_isValidIndex(t){return this._items?!!this._items.toArray()[t]:!0}_setTabFocus(t){if(this._showPaginationControls&&this._scrollToLabel(t),this._items&&this._items.length){this._items.toArray()[t].focus();let e=this._tabListContainer.nativeElement;this._getLayoutDirection()==`ltr`?e.scrollLeft=0:e.scrollLeft=e.scrollWidth-e.offsetWidth}}_getLayoutDirection(){return this._dir&&this._dir.value===`rtl`?`rtl`:`ltr`}_updateTabScrollPosition(){if(this.disablePagination)return;let t=this.scrollDistance,e=this._getLayoutDirection()===`ltr`?-t:t;this._tabList.nativeElement.style.transform=`translateX(${Math.round(e)}px)`,(this._platform.TRIDENT||this._platform.EDGE)&&(this._tabListContainer.nativeElement.scrollLeft=0)}get scrollDistance(){return this._scrollDistance}set scrollDistance(t){this._scrollTo(t)}_scrollHeader(t){let e=this._tabListContainer.nativeElement.offsetWidth,a=(t==`before`?-1:1)*e/3;return this._scrollTo(this._scrollDistance+a)}_handlePaginatorClick(t){this._stopInterval(),this._scrollHeader(t)}_scrollToLabel(t){if(this.disablePagination)return;let e=this._items?this._items.toArray()[t]:null;if(!e)return;let a=this._tabListContainer.nativeElement.offsetWidth,{offsetLeft:o,offsetWidth:s}=e.elementRef.nativeElement,r,f;this._getLayoutDirection()==`ltr`?(r=o,f=r+s):(f=this._tabListInner.nativeElement.offsetWidth-o,r=f-s);let v=this.scrollDistance,d=this.scrollDistance+a;r<v?this.scrollDistance-=v-r:f>d&&(this.scrollDistance+=Math.min(f-d,r-v))}_checkPaginationEnabled(){if(this.disablePagination)this._showPaginationControls=!1;else{let a=this._tabListInner.nativeElement.scrollWidth-this._elementRef.nativeElement.offsetWidth>=5;a||(this.scrollDistance=0),a!==this._showPaginationControls&&(this._showPaginationControls=a,this._changeDetectorRef.markForCheck())}}_checkScrollingControls(){this.disablePagination?this._disableScrollAfter=this._disableScrollBefore=!0:(this._disableScrollBefore=this.scrollDistance==0,this._disableScrollAfter=this.scrollDistance==this._getMaxScrollDistance(),this._changeDetectorRef.markForCheck())}_getMaxScrollDistance(){return this._tabListInner.nativeElement.scrollWidth-this._tabListContainer.nativeElement.offsetWidth||0}_alignInkBarToSelectedTab(){let t=this._items&&this._items.length?this._items.toArray()[this.selectedIndex]:null,e=t?t.elementRef.nativeElement:null;e?this._inkBar.alignToElement(e):this._inkBar.hide()}_stopInterval(){this._stopScrolling.next()}_handlePaginatorPress(t,e){e&&e.button!=null&&e.button!==0||(this._stopInterval(),Pl(je,He).pipe(Zn(xI(this._stopScrolling,this._destroyed))).subscribe(()=>{let{maxScrollDistance:a,distance:o}=this._scrollHeader(t);(o===0||o>=a)&&this._stopInterval()}))}_scrollTo(t){if(this.disablePagination)return{maxScrollDistance:0,distance:0};let e=this._getMaxScrollDistance();return this._scrollDistance=Math.max(0,Math.min(e,t)),this._scrollDistanceChanged=!0,this._checkScrollingControls(),{maxScrollDistance:e,distance:this._scrollDistance}}static ɵfac=function(e){return new(e||i)};static ɵdir=Be({type:i,inputs:{disablePagination:[2,`disablePagination`,`disablePagination`,xe$1],selectedIndex:[2,`selectedIndex`,`selectedIndex`,kf]},outputs:{selectFocusedIndex:`selectFocusedIndex`,indexFocused:`indexFocused`}})}return i})();var Ue=new v(`MAT_TABS_CONFIG`);var jt=(()=>{class i extends $e{_focusedItem=B(null);get fitInkBarToContent(){return this._fitInkBarToContent.value}set fitInkBarToContent(t){this._fitInkBarToContent.next(t),this._changeDetectorRef.markForCheck()}_fitInkBarToContent=new pe(!1);stretchTabs=!0;animationDuration=``;_items;get backgroundColor(){return this._backgroundColor}set backgroundColor(t){let e=this._elementRef.nativeElement.classList;e.remove(`mat-tabs-with-background`,`mat-background-${this.backgroundColor}`),t&&e.add(`mat-tabs-with-background`,`mat-background-${t}`),this._backgroundColor=t}_backgroundColor;get disableRipple(){return this._disableRipple()}set disableRipple(t){this._disableRipple.set(t)}_disableRipple=B(!1);color=`primary`;tabPanel;_tabListContainer;_tabList;_tabListInner;_nextPaginator;_previousPaginator;_inkBar;constructor(){let t=m(Ue,{optional:!0});super(),this.disablePagination=t&&t.disablePagination!=null?t.disablePagination:!1,this.fitInkBarToContent=t&&t.fitInkBarToContent!=null?t.fitInkBarToContent:!1,this.stretchTabs=t&&t.stretchTabs!=null?t.stretchTabs:!0}_itemSelected(){}ngAfterContentInit(){this._inkBar=new Gt(this._items),this._items.changes.pipe(jl(null),Zn(this._destroyed)).subscribe(()=>this.updateActiveLink()),super.ngAfterContentInit(),this._keyManager.change.pipe(jl(null),Zn(this._destroyed)).subscribe(()=>this._focusedItem.set(this._keyManager?.activeItem||null))}ngAfterViewInit(){this.tabPanel,super.ngAfterViewInit()}updateActiveLink(){if(!this._items)return;let t=this._items.toArray();for(let e=0;e<t.length;e++)if(t[e].active){this.selectedIndex=e,this.tabPanel&&(this.tabPanel._activeTabId=t[e].id),this._focusedItem.set(t[e]),this._changeDetectorRef.markForCheck();return}this.selectedIndex=-1}_getRole(){return this.tabPanel?`tablist`:this._elementRef.nativeElement.getAttribute(`role`)}_hasFocus(t){return this._keyManager?.activeItem===t}static ɵfac=function(e){return new(e||i)};static ɵcmp=(function(){let t=[`tabListContainer`],e=[`tabList`],a=[`tabListInner`],o=[`nextPaginator`],s=[`previousPaginator`];return Ve({type:i,selectors:[[``,`mat-tab-nav-bar`,``]],contentQueries:function(v,d,g){if(v&1&&bc(g,Ht,5),v&2){let R;yf(R=Ef())&&(d._items=R)}},viewQuery:function(v,d){if(v&1&&hv(t,7)(e,7)(a,7)(o,5)(s,5),v&2){let g;yf(g=Ef())&&(d._tabListContainer=g.first),yf(g=Ef())&&(d._tabList=g.first),yf(g=Ef())&&(d._tabListInner=g.first),yf(g=Ef())&&(d._nextPaginator=g.first),yf(g=Ef())&&(d._previousPaginator=g.first)}},hostAttrs:[1,`mat-mdc-tab-nav-bar`,`mat-mdc-tab-header`],hostVars:17,hostBindings:function(v,d){v&2&&(Tn(`role`,d._getRole()),yv(`--%NS%mat-tab-header-animation-duration`,d.animationDuration),Ot(`mat-mdc-tab-header-pagination-controls-enabled`,d._showPaginationControls)(`mat-mdc-tab-header-rtl`,d._getLayoutDirection()==`rtl`)(`mat-mdc-tab-nav-bar-stretch-tabs`,d.stretchTabs)(`mat-primary`,d.color!==`warn`&&d.color!==`accent`)(`mat-accent`,d.color===`accent`)(`mat-warn`,d.color===`warn`)(`_mat-animation-noopable`,d._animationsDisabled))},inputs:{fitInkBarToContent:[2,`fitInkBarToContent`,`fitInkBarToContent`,xe$1],stretchTabs:[2,`mat-stretch-tabs`,`stretchTabs`,xe$1],animationDuration:[2,`animationDuration`,`animationDuration`,ze],backgroundColor:`backgroundColor`,disableRipple:[2,`disableRipple`,`disableRipple`,xe$1],color:`color`,tabPanel:`tabPanel`},exportAs:[`matTabNavBar`,`matTabNav`],features:[bo],ngContentSelectors:[`*`],decls:13,vars:6,consts:[[`previousPaginator`,``],[`tabListContainer`,``],[`tabList`,``],[`tabListInner`,``],[`nextPaginator`,``],[`mat-ripple`,``,1,`mat-mdc-tab-header-pagination`,`mat-mdc-tab-header-pagination-before`,3,`click`,`mousedown`,`touchend`,`matRippleDisabled`],[1,`mat-mdc-tab-header-pagination-chevron`],[1,`mat-mdc-tab-link-container`,3,`keydown`],[1,`mat-mdc-tab-list`,3,`cdkObserveContent`],[1,`mat-mdc-tab-links`],[`mat-ripple`,``,1,`mat-mdc-tab-header-pagination`,`mat-mdc-tab-header-pagination-after`,3,`mousedown`,`click`,`touchend`,`matRippleDisabled`]],template:function(v,d){v&1&&(yr(),rc(0,`div`,5,0),gc(`click`,function(){return d._handlePaginatorClick(`before`)})(`mousedown`,function(R){return d._handlePaginatorPress(`before`,R)})(`touchend`,function(){return d._stopInterval()}),hc(2,`div`,6),vf(),rc(3,`div`,7,1),gc(`keydown`,function(R){return d._handleKeydown(R)}),rc(5,`div`,8,2),gc(`cdkObserveContent`,function(){return d._onContentChanges()}),rc(7,`div`,9,3),Et(9),vf()()(),rc(10,`div`,10,4),gc(`mousedown`,function(R){return d._handlePaginatorPress(`after`,R)})(`click`,function(){return d._handlePaginatorClick(`after`)})(`touchend`,function(){return d._stopInterval()}),hc(12,`div`,6),vf()),v&2&&(Ot(`mat-mdc-tab-header-pagination-disabled`,d._disableScrollBefore),uv(`matRippleDisabled`,d._disableScrollBefore||d.disableRipple),Mi(10),Ot(`mat-mdc-tab-header-pagination-disabled`,d._disableScrollAfter),uv(`matRippleDisabled`,d._disableScrollAfter||d.disableRipple))},dependencies:[mG,sq],styles:[`.mdc-tab {
  min-width: 90px;
  padding: 0 24px;
  display: flex;
  flex: 1 0 auto;
  justify-content: center;
  box-sizing: border-box;
  border: none;
  outline: none;
  text-align: center;
  white-space: nowrap;
  cursor: pointer;
  z-index: 1;
  touch-action: manipulation;
}

.mdc-tab__content {
  display: flex;
  align-items: center;
  justify-content: center;
  height: inherit;
  pointer-events: none;
}

.mdc-tab__text-label {
  transition: 150ms color linear;
  display: inline-block;
  line-height: 1;
  z-index: 2;
}

.mdc-tab--active .mdc-tab__text-label {
  transition-delay: 100ms;
}

._mat-animation-noopable .mdc-tab__text-label {
  transition: none;
}

.mdc-tab-indicator {
  display: flex;
  position: absolute;
  top: 0;
  left: 0;
  justify-content: center;
  width: 100%;
  height: 100%;
  pointer-events: none;
  z-index: 1;
}

.mdc-tab-indicator__content {
  transition: var(--%NS%mat-tab-header-animation-duration, 250ms) transform cubic-bezier(0.4, 0, 0.2, 1);
  transform-origin: left;
  opacity: 0;
}

.mdc-tab-indicator__content--underline {
  align-self: flex-end;
  box-sizing: border-box;
  width: 100%;
  border-top-style: solid;
}

.mdc-tab-indicator--active .mdc-tab-indicator__content {
  opacity: 1;
}

._mat-animation-noopable .mdc-tab-indicator__content, .mdc-tab-indicator--no-transition .mdc-tab-indicator__content {
  transition: none;
}

.mat-mdc-tab-ripple.mat-mdc-tab-ripple {
  position: absolute;
  top: 0;
  left: 0;
  bottom: 0;
  right: 0;
  pointer-events: none;
}

.mat-mdc-tab-header {
  display: flex;
  overflow: hidden;
  position: relative;
  flex-shrink: 0;
}

.mdc-tab-indicator .mdc-tab-indicator__content {
  transition-duration: var(--%NS%mat-tab-header-animation-duration, 250ms);
}

.mat-mdc-tab-header-pagination {
  -webkit-user-select: none;
  user-select: none;
  position: relative;
  display: none;
  justify-content: center;
  align-items: center;
  min-width: 32px;
  cursor: pointer;
  z-index: 2;
  -webkit-tap-highlight-color: transparent;
  touch-action: none;
  box-sizing: content-box;
  outline: 0;
}
.mat-mdc-tab-header-pagination::-moz-focus-inner {
  border: 0;
}
.mat-mdc-tab-header-pagination .mat-ripple-element {
  opacity: 0.12;
  background-color: var(--%NS%mat-tab-inactive-ripple-color, var(--%NS%mat-sys-on-surface));
}
.mat-mdc-tab-header-pagination-controls-enabled .mat-mdc-tab-header-pagination {
  display: flex;
}

.mat-mdc-tab-header-pagination-before,
.mat-mdc-tab-header-rtl .mat-mdc-tab-header-pagination-after {
  padding-left: 4px;
}
.mat-mdc-tab-header-pagination-before .mat-mdc-tab-header-pagination-chevron,
.mat-mdc-tab-header-rtl .mat-mdc-tab-header-pagination-after .mat-mdc-tab-header-pagination-chevron {
  transform: rotate(-135deg);
}

.mat-mdc-tab-header-rtl .mat-mdc-tab-header-pagination-before,
.mat-mdc-tab-header-pagination-after {
  padding-right: 4px;
}
.mat-mdc-tab-header-rtl .mat-mdc-tab-header-pagination-before .mat-mdc-tab-header-pagination-chevron,
.mat-mdc-tab-header-pagination-after .mat-mdc-tab-header-pagination-chevron {
  transform: rotate(45deg);
}

.mat-mdc-tab-header-pagination-chevron {
  border-style: solid;
  border-width: 2px 2px 0 0;
  height: 8px;
  width: 8px;
  border-color: var(--%NS%mat-tab-pagination-icon-color, var(--%NS%mat-sys-on-surface));
}

.mat-mdc-tab-header-pagination-disabled {
  box-shadow: none;
  cursor: default;
  pointer-events: none;
}
.mat-mdc-tab-header-pagination-disabled .mat-mdc-tab-header-pagination-chevron {
  opacity: 0.4;
}

.mat-mdc-tab-list {
  flex-grow: 1;
  position: relative;
  transition: transform 500ms cubic-bezier(0.35, 0, 0.25, 1);
}
._mat-animation-noopable .mat-mdc-tab-list {
  transition: none;
}

.mat-mdc-tab-links {
  display: flex;
  flex: 1 0 auto;
}
[mat-align-tabs=center] > .mat-mdc-tab-link-container .mat-mdc-tab-links {
  justify-content: center;
}
[mat-align-tabs=end] > .mat-mdc-tab-link-container .mat-mdc-tab-links {
  justify-content: flex-end;
}
.cdk-drop-list .mat-mdc-tab-links, .mat-mdc-tab-links.cdk-drop-list {
  min-height: var(--%NS%mat-tab-container-height, 48px);
}

.mat-mdc-tab-link-container {
  display: flex;
  flex-grow: 1;
  overflow: hidden;
  z-index: 1;
  border-bottom-style: solid;
  border-bottom-width: var(--%NS%mat-tab-divider-height, 1px);
  border-bottom-color: var(--%NS%mat-tab-divider-color, var(--%NS%mat-sys-surface-variant));
}

.mat-mdc-tab-nav-bar.mat-tabs-with-background > .mat-mdc-tab-link-container, .mat-mdc-tab-nav-bar.mat-tabs-with-background > .mat-mdc-tab-header-pagination {
  background-color: var(--%NS%mat-tab-background-color);
}
.mat-mdc-tab-nav-bar.mat-tabs-with-background.mat-primary > .mat-mdc-tab-link-container .mat-mdc-tab-link .mdc-tab__text-label {
  color: var(--%NS%mat-tab-foreground-color);
}
.mat-mdc-tab-nav-bar.mat-tabs-with-background.mat-primary > .mat-mdc-tab-link-container .mdc-tab-indicator__content--underline {
  border-color: var(--%NS%mat-tab-foreground-color);
}
.mat-mdc-tab-nav-bar.mat-tabs-with-background:not(.mat-primary) > .mat-mdc-tab-link-container .mat-mdc-tab-link:not(.mdc-tab--active) .mdc-tab__text-label {
  color: var(--%NS%mat-tab-foreground-color);
}
.mat-mdc-tab-nav-bar.mat-tabs-with-background:not(.mat-primary) > .mat-mdc-tab-link-container .mat-mdc-tab-link:not(.mdc-tab--active) .mdc-tab-indicator__content--underline {
  border-color: var(--%NS%mat-tab-foreground-color);
}
.mat-mdc-tab-nav-bar.mat-tabs-with-background > .mat-mdc-tab-link-container .mat-mdc-tab-header-pagination-chevron,
.mat-mdc-tab-nav-bar.mat-tabs-with-background > .mat-mdc-tab-link-container .mat-focus-indicator::before, .mat-mdc-tab-nav-bar.mat-tabs-with-background > .mat-mdc-tab-header-pagination .mat-mdc-tab-header-pagination-chevron,
.mat-mdc-tab-nav-bar.mat-tabs-with-background > .mat-mdc-tab-header-pagination .mat-focus-indicator::before {
  border-color: var(--%NS%mat-tab-foreground-color);
}
.mat-mdc-tab-nav-bar.mat-tabs-with-background > .mat-mdc-tab-link-container .mat-ripple-element, .mat-mdc-tab-nav-bar.mat-tabs-with-background > .mat-mdc-tab-link-container .mdc-tab__ripple::before, .mat-mdc-tab-nav-bar.mat-tabs-with-background > .mat-mdc-tab-header-pagination .mat-ripple-element, .mat-mdc-tab-nav-bar.mat-tabs-with-background > .mat-mdc-tab-header-pagination .mdc-tab__ripple::before {
  background-color: var(--%NS%mat-tab-foreground-color);
}
.mat-mdc-tab-nav-bar.mat-tabs-with-background > .mat-mdc-tab-link-container .mat-mdc-tab-header-pagination-chevron, .mat-mdc-tab-nav-bar.mat-tabs-with-background > .mat-mdc-tab-header-pagination .mat-mdc-tab-header-pagination-chevron {
  color: var(--%NS%mat-tab-foreground-color);
}
`],encapsulation:2,changeDetection:1})})()}return i})();var Ht=(()=>{class i extends Ge{_tabNavBar=m(jt);elementRef=m(de);_focusMonitor=m(Lp);_destroyed=new H;_isActive=!1;_tabIndex=Le$1(()=>this._tabNavBar._focusedItem()===this?this.tabIndex:-1);get active(){return this._isActive}set active(t){t!==this._isActive&&(this._isActive=t,this._tabNavBar.updateActiveLink())}disabled=!1;get disableRipple(){return this._disableRipple()}set disableRipple(t){this._disableRipple.set(t)}_disableRipple=B(!1);tabIndex=0;rippleConfig;get rippleDisabled(){return this.disabled||this.disableRipple||this._tabNavBar.disableRipple||!!this.rippleConfig.disabled}id=m(Up).getId(`mat-tab-link-`);constructor(){super(),m(Ji).load(YE);let t=m(Bp,{optional:!0}),e=m(new yo(`tabindex`),{optional:!0});this.rippleConfig=t||{},this.tabIndex=e==null?0:parseInt(e)||0,Vo()&&(this.rippleConfig.animation={enterDuration:0,exitDuration:0}),this._tabNavBar._fitInkBarToContent.pipe(Zn(this._destroyed)).subscribe(a=>{this.fitInkBarToContent=a})}focus(){this.elementRef.nativeElement.focus()}ngAfterViewInit(){this._focusMonitor.monitor(this.elementRef)}ngOnDestroy(){this._destroyed.next(),this._destroyed.complete(),super.ngOnDestroy(),this._focusMonitor.stopMonitoring(this.elementRef)}_handleFocus(){this._tabNavBar.focusIndex=this._tabNavBar._items.toArray().indexOf(this)}_handleKeydown(t){(t.keyCode===32||t.keyCode===13)&&(this.disabled?t.preventDefault():this._tabNavBar.tabPanel&&(t.keyCode===32&&t.preventDefault(),this.elementRef.nativeElement.click()))}_getAriaControls(){return this._tabNavBar.tabPanel?this._tabNavBar.tabPanel?.id:this.elementRef.nativeElement.getAttribute(`aria-controls`)}_getAriaSelected(){return this._tabNavBar.tabPanel?this.active?`true`:`false`:this.elementRef.nativeElement.getAttribute(`aria-selected`)}_getAriaCurrent(){return this.active&&!this._tabNavBar.tabPanel?`page`:null}_getRole(){return this._tabNavBar.tabPanel?`tab`:this.elementRef.nativeElement.getAttribute(`role`)}static ɵfac=function(e){return new(e||i)};static ɵcmp=(function(){return Ve({type:i,selectors:[[``,`mat-tab-link`,``],[``,`matTabLink`,``]],hostAttrs:[1,`mdc-tab`,`mat-mdc-tab-link`,`mat-focus-indicator`],hostVars:11,hostBindings:function(a,o){a&1&&gc(`focus`,function(){return o._handleFocus()})(`keydown`,function(r){return o._handleKeydown(r)}),a&2&&(Tn(`aria-controls`,o._getAriaControls())(`aria-current`,o._getAriaCurrent())(`aria-disabled`,o.disabled)(`aria-selected`,o._getAriaSelected())(`id`,o.id)(`tabIndex`,o._tabIndex())(`role`,o._getRole()),Ot(`mat-mdc-tab-disabled`,o.disabled)(`mdc-tab--active`,o.active))},inputs:{active:[2,`active`,`active`,xe$1],disabled:[2,`disabled`,`disabled`,xe$1],disableRipple:[2,`disableRipple`,`disableRipple`,xe$1],tabIndex:[2,`tabIndex`,`tabIndex`,e=>e==null?0:kf(e)],id:`id`},exportAs:[`matTabLink`],features:[bo],ngContentSelectors:[`*`],decls:5,vars:2,consts:[[1,`mdc-tab__ripple`],[`mat-ripple`,``,1,`mat-mdc-tab-ripple`,3,`matRippleTrigger`,`matRippleDisabled`],[1,`mdc-tab__content`],[1,`mdc-tab__text-label`]],template:function(a,o){a&1&&(yr(),hc(0,`span`,0)(1,`div`,1),rc(2,`span`,2)(3,`span`,3),Et(4),vf()()),a&2&&(Mi(),uv(`matRippleTrigger`,o.elementRef.nativeElement)(`matRippleDisabled`,o.rippleDisabled))},dependencies:[mG],styles:[`.mat-mdc-tab-link {
  -webkit-tap-highlight-color: transparent;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  text-decoration: none;
  background: none;
  height: var(--%NS%mat-tab-container-height, 48px);
  font-family: var(--%NS%mat-tab-label-text-font, var(--%NS%mat-sys-title-small-font));
  font-size: var(--%NS%mat-tab-label-text-size, var(--%NS%mat-sys-title-small-size));
  letter-spacing: var(--%NS%mat-tab-label-text-tracking, var(--%NS%mat-sys-title-small-tracking));
  line-height: var(--%NS%mat-tab-label-text-line-height, var(--%NS%mat-sys-title-small-line-height));
  font-weight: var(--%NS%mat-tab-label-text-weight, var(--%NS%mat-sys-title-small-weight));
}
.mat-mdc-tab-link.mdc-tab {
  flex-grow: 0;
}
.mat-mdc-tab-link .mdc-tab-indicator__content--underline {
  border-color: var(--%NS%mat-tab-active-indicator-color, var(--%NS%mat-sys-primary));
  border-top-width: var(--%NS%mat-tab-active-indicator-height, 2px);
  border-radius: var(--%NS%mat-tab-active-indicator-shape, 0);
}
.mat-mdc-tab-link:hover .mdc-tab__text-label {
  color: var(--%NS%mat-tab-inactive-hover-label-text-color, var(--%NS%mat-sys-on-surface));
}
.mat-mdc-tab-link:focus .mdc-tab__text-label {
  color: var(--%NS%mat-tab-inactive-focus-label-text-color, var(--%NS%mat-sys-on-surface));
}
.mat-mdc-tab-link.mdc-tab--active .mdc-tab__text-label {
  color: var(--%NS%mat-tab-active-label-text-color, var(--%NS%mat-sys-on-surface));
}
.mat-mdc-tab-link.mdc-tab--active .mdc-tab__ripple::before,
.mat-mdc-tab-link.mdc-tab--active .mat-ripple-element {
  background-color: var(--%NS%mat-tab-active-ripple-color, var(--%NS%mat-sys-on-surface));
}
.mat-mdc-tab-link.mdc-tab--%NS%active:hover .mdc-tab__text-label {
  color: var(--%NS%mat-tab-active-hover-label-text-color, var(--%NS%mat-sys-on-surface));
}
.mat-mdc-tab-link.mdc-tab--%NS%active:hover .mdc-tab-indicator__content--underline {
  border-color: var(--%NS%mat-tab-active-hover-indicator-color, var(--%NS%mat-sys-primary));
}
.mat-mdc-tab-link.mdc-tab--%NS%active:focus .mdc-tab__text-label {
  color: var(--%NS%mat-tab-active-focus-label-text-color, var(--%NS%mat-sys-on-surface));
}
.mat-mdc-tab-link.mdc-tab--%NS%active:focus .mdc-tab-indicator__content--underline {
  border-color: var(--%NS%mat-tab-active-focus-indicator-color, var(--%NS%mat-sys-primary));
}
.mat-mdc-tab-link.mat-mdc-tab-disabled {
  opacity: 0.4;
  pointer-events: none;
}
.mat-mdc-tab-link.mat-mdc-tab-disabled .mdc-tab__content {
  pointer-events: none;
}
.mat-mdc-tab-link.mat-mdc-tab-disabled .mdc-tab__ripple::before,
.mat-mdc-tab-link.mat-mdc-tab-disabled .mat-ripple-element {
  background-color: var(--%NS%mat-tab-disabled-ripple-color, var(--%NS%mat-sys-on-surface-variant));
}
.mat-mdc-tab-link .mdc-tab__ripple::before {
  content: "";
  display: block;
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  opacity: 0;
  pointer-events: none;
  background-color: var(--%NS%mat-tab-inactive-ripple-color, var(--%NS%mat-sys-on-surface));
}
.mat-mdc-tab-link .mdc-tab__text-label {
  color: var(--%NS%mat-tab-inactive-label-text-color, var(--%NS%mat-sys-on-surface));
  display: inline-flex;
  align-items: center;
}
.mat-mdc-tab-link .mdc-tab__content {
  position: relative;
  pointer-events: auto;
}
.mat-mdc-tab-link:hover .mdc-tab__ripple::before {
  opacity: 0.04;
}
.mat-mdc-tab-link.cdk-program-focused .mdc-tab__ripple::before, .mat-mdc-tab-link.cdk-keyboard-focused .mdc-tab__ripple::before {
  opacity: 0.12;
}
.mat-mdc-tab-link .mat-ripple-element {
  opacity: 0.12;
  background-color: var(--%NS%mat-tab-inactive-ripple-color, var(--%NS%mat-sys-on-surface));
}
.mat-mdc-tab-header.mat-mdc-tab-nav-bar-stretch-tabs .mat-mdc-tab-link {
  flex-grow: 1;
}
.mat-mdc-tab-link::before {
  margin: 5px;
}

@media (max-width: 599px) {
  .mat-mdc-tab-link {
    min-width: 72px;
  }
}
`],encapsulation:2})})()}return i})();var Ie=(()=>{class i{id=m(Up).getId(`mat-tab-nav-panel-`);_activeTabId;static ɵfac=function(e){return new(e||i)};static ɵcmp=(function(){return Ve({type:i,selectors:[[`mat-tab-nav-panel`]],hostAttrs:[`role`,`tabpanel`,1,`mat-mdc-tab-nav-panel`],hostVars:2,hostBindings:function(a,o){a&2&&Tn(`aria-labelledby`,o._activeTabId)(`id`,o.id)},inputs:{id:`id`},exportAs:[`matTabNavPanel`],ngContentSelectors:[`*`],decls:1,vars:0,template:function(a,o){a&1&&(yr(),Et(0))},encapsulation:2})})()}return i})();var We=`redoc.colorMode`;var qe=[`light`,`dark`];var yt=class i{document=m(U);http=m(Xi);loading;async render(n,t,e){let[a,o]=await Promise.all([this.load(),EI(this.http.get(n,{responseType:`text`}))]),s=a.init(o,e,t);return{destroy:()=>s.unmount()}}setColorMode(n){let t=this.document.documentElement;t.classList.remove(...qe),t.classList.add(n);try{localStorage.setItem(We,n)}catch{}}load(){return this.loading??=import(`./chunk-DycSoky0.js`).catch(n=>{throw this.loading=void 0,n}),this.loading}static ɵfac=function(t){return new(t||i)};static ɵprov=O({token:i,factory:i.ɵfac})};var Qe=[`host`];var Ye=()=>({view:`source`});function Ze(i,n){if(i&1&&(rc(0,`p`,1),PC(1),vf()),i&2){let t=uC();Mi(),If(`Loading the `,t.api().name,` reference…`)}}function Ke(i,n){i&1&&(rc(0,`p`,2),PC(1,` The interactive reference could not be loaded. `),rc(2,`a`,4),PC(3,`Read the specification source`),vf(),PC(4,` instead. `),vf()),i&2&&(Mi(2),uv(`queryParams`,$C(1,Ye)))}var Ct=class i{redoc=m(yt);theme=m(z);host=_$.required(`host`);api=Io.required();scrollOffset=Io(()=>0);status=B(`loading`);run=0;rendered;constructor(){no(()=>this.redoc.setColorMode(this.theme.scheme())),D$({write:()=>{let n=this.api(),t=this.host().nativeElement;L(()=>this.render(n,t))}}),m(ye$1).onDestroy(()=>{this.run++,this.clear()})}async render(n,t){let e=++this.run;this.status.set(`loading`),this.clear();let a=t.ownerDocument.createElement(`div`);t.replaceChildren(a);try{let o=await this.redoc.render(g8(n),a,{scrollYOffset:this.scrollOffset(),hideDownloadButtons:!0,disableTelemetry:!0});if(e!==this.run){o.destroy();return}this.rendered=o,this.status.set(`ready`)}catch(o){console.error(o),e===this.run&&this.status.set(`failed`)}}clear(){this.rendered?.destroy(),this.rendered=void 0}static ɵfac=function(t){return new(t||i)};static ɵcmp=Ve({type:i,selectors:[[`app-redoc-view`]],viewQuery:function(t,e){t&1&&gv(e.host,Qe,5),t&2&&pC()},inputs:{api:[1,`api`],scrollOffset:[1,`scrollOffset`]},decls:4,vars:2,consts:[[`host`,``],[`role`,`status`,1,`status`],[`role`,`alert`,1,`status`],[1,`redoc`,3,`hidden`],[`routerLink`,`.`,3,`queryParams`]],template:function(t,e){if(t&1&&(Fi(0,Ze,2,1,`p`,1)(1,Ke,5,2,`p`,2),hc(2,`div`,3,0)),t&2){let a;ji((a=e.status())===`loading`?0:a===`failed`?1:-1),Mi(2),uv(`hidden`,e.status()===`failed`)}},dependencies:[ul],styles:[`[_nghost-%COMP%]{display:block}.status[_ngcontent-%COMP%]{max-width:72rem;margin:2rem auto;padding-inline:1rem;color:var(--%NS%mat-sys-on-surface-variant)}`]})};var Xe=/^(\s*)(#.*)$/;var Je=/^(\s*(?:-\s+)?)([A-Za-z0-9_$/{}.\-"']+)(:)(?=\s|$)/;var tn=/^(\s+)("[^"]*"|'[^']*'|-?\d+(?:\.\d+)?)\s*$/;function Ae(i){let n=Xe.exec(i);if(n)return Te([{kind:`plain`,text:n[1]},{kind:`comment`,text:n[2]}]);let t=[],e=i,a=Je.exec(e);if(a)t.push({kind:`plain`,text:a[1]},{kind:`key`,text:a[2]}),t.push({kind:`plain`,text:a[3]}),e=e.slice(a[0].length);else{let s=/^(\s*-)(?=\s)/.exec(e);s&&(t.push({kind:`plain`,text:s[1]}),e=e.slice(s[0].length))}let o=tn.exec(e);if(o&&(a||t.length>0)){let s=/^["']/.test(o[2])?`string`:`number`;t.push({kind:`plain`,text:o[1]},{kind:s,text:o[2]}),e=e.slice(o[1].length+o[2].length)}return t.push({kind:`plain`,text:e}),Te(t)}function Te(i){return i.filter(n=>n.text!==``)}function en(i,n){if(i&1&&PC(0),i&2){let t=uC();If(` · `,t.lines().length,` lines `)}}function nn(i,n){if(i&1&&hc(0,`app-copy-button`,2),i&2){let t=uC();uv(`text`,t.spec.value())}}function an(i,n){i&1&&(rc(0,`p`,3),PC(1,`The specification could not be loaded.`),vf())}function on(i,n){if(i&1&&(rc(0,`p`,4),PC(1),vf()),i&2){let t=uC();Mi(),If(`Loading openapi/`,t.api().file,`…`)}}function rn(i,n){if(i&1&&(rc(0,`span`),PC(1),vf()),i&2){let t=n.$implicit;Hi(`tok-`+t.kind),Mi(),Dv(t.text)}}function sn(i,n){if(i&1&&(rc(0,`span`,6),tC(1,rn,2,3,`span`,7,JD),vf(),PC(3,`
`)),i&2){let t=n.$implicit;Mi(),nC(t)}}function ln(i,n){if(i&1&&(rc(0,`pre`,5)(1,`code`),tC(2,sn,4,0,null,null,JD),vf()()),i&2){let t=uC();Mi(2),nC(t.lines())}}var xt=class i{api=Io.required();spec=J2.text(()=>g8(this.api()));lines=Le$1(()=>(this.spec.hasValue()?this.spec.value():``).replace(/\n$/,``).split(`
`).map(Ae));static ɵfac=function(t){return new(t||i)};static ɵcmp=Ve({type:i,selectors:[[`app-spec-source`]],inputs:{api:[1,`api`]},decls:9,vars:5,consts:[[1,`code-panel`],[1,`code-panel-head`],[`subject`,`the specification`,3,`text`],[`role`,`alert`,1,`status`],[`role`,`status`,1,`status`],[`tabindex`,`0`],[1,`line`],[3,`class`]],template:function(t,e){t&1&&(rc(0,`section`,0)(1,`div`,1)(2,`span`),PC(3),Fi(4,en,1,1),vf(),Fi(5,nn,1,1,`app-copy-button`,2),vf(),Fi(6,an,2,0,`p`,3)(7,on,2,1,`p`,4)(8,ln,4,0,`pre`,5),vf()),t&2&&(Tn(`aria-label`,`Source of openapi/`+e.api().file),Mi(3),If(` openapi/`,e.api().file,` `),Mi(),ji(e.spec.hasValue()?4:-1),Mi(),ji(e.spec.hasValue()?5:-1),Mi(),ji(e.spec.error()?6:e.spec.isLoading()?7:8))},dependencies:[k],styles:[`[_nghost-%COMP%]{display:block;max-width:72rem;margin:1.5rem auto;padding-inline:1rem}.status[_ngcontent-%COMP%]{margin:0;padding:1rem 1.25rem}pre[_ngcontent-%COMP%]{counter-reset:line}.line[_ngcontent-%COMP%]{counter-increment:line}.line[_ngcontent-%COMP%]:before{display:inline-block;width:3.5ch;margin-inline-end:1.5ch;color:var(--%NS%code-muted);content:counter(line);text-align:end;-webkit-user-select:none;user-select:none}`]})};var dn=[`bar`];var cn=i=>[`/api`,i];var un=(i,n)=>n.id;var mn=(i,n)=>n.href;function pn(i,n){if(i&1&&(rc(0,`a`,4),PC(1),vf()),i&2){let t=n.$implicit,e=uC();uv(`routerLink`,zC(3,cn,t.id))(`active`,t.id===e.current().id),Mi(),If(` `,t.shortName,` `)}}function bn(i,n){if(i&1&&(rc(0,`a`,18),PC(1),vf(),PC(2)),i&2){let t=n.$implicit,e=n.$index,a=n.$count;uv(`href`,t.href,nb),Mi(),Dv(t.label),Mi(),If(``,e===a-1?``:`,`,` `)}}function hn(i,n){if(i&1&&hc(0,`app-spec-source`,16),i&2){let t=uC();uv(`api`,t.current())}}function gn(i,n){if(i&1&&hc(0,`app-redoc-view`,17),i&2){let t=uC();uv(`api`,t.current())(`scrollOffset`,t.scrollOffset)}}var Ne=class i{router=m(Mr);bar=_$.required(`bar`);api=Io.required();view=Io();apis=OM;current=Le$1(()=>h8(this.api())??OM[0]);currentView=Le$1(()=>this.view()===`source`?`source`:`reference`);specUrl=Le$1(()=>g8(this.current()));githubUrl=Le$1(()=>c8(g8(this.current())));scrollOffset=()=>this.bar().nativeElement.getBoundingClientRect().bottom;setView(n){this.router.navigate([],{queryParams:{view:n===`source`?`source`:null},queryParamsHandling:`merge`,replaceUrl:!0})}static ɵfac=function(t){return new(t||i)};static ɵcmp=Ve({type:i,selectors:[[`app-api-reference`]],viewQuery:function(t,e){t&1&&gv(e.bar,dn,5),t&2&&pC()},inputs:{api:[1,`api`],view:[1,`view`]},decls:33,vars:8,consts:[[`bar`,``],[`panel`,``],[1,`api-bar`],[`mat-tab-nav-bar`,``,`mat-stretch-tabs`,`false`,`mat-align-tabs`,`start`,`aria-label`,`APIs`,1,`tabs`,3,`tabPanel`],[`mat-tab-link`,``,`queryParamsHandling`,`preserve`,3,`routerLink`,`active`],[1,`spec-bar`],[1,`spec-title`],[1,`guides`],[1,`spacer`],[`aria-label`,`View`,`hideSingleSelectionIndicator`,``,3,`change`,`value`],[`value`,`reference`],[`value`,`source`],[`matButton`,``,3,`href`],[`aria-hidden`,`true`],[`matButton`,``,1,`github`,3,`href`],[1,`panel`],[3,`api`],[3,`api`,`scrollOffset`],[3,`href`]],template:function(t,e){if(t&1&&(rc(0,`div`,2,0)(2,`nav`,3),tC(3,pn,2,5,`a`,4,un),vf(),rc(5,`div`,5)(6,`div`,6)(7,`h1`),PC(8),vf(),rc(9,`code`),PC(10),vf()(),rc(11,`p`,7),PC(12,` Guides: `),tC(13,bn,3,3,null,null,mn),vf(),hc(15,`span`,8),rc(16,`mat-button-toggle-group`,9),gc(`change`,function(o){return e.setView(o.value)}),rc(17,`mat-button-toggle`,10),PC(18,`Reference`),vf(),rc(19,`mat-button-toggle`,11),PC(20,`Source`),vf()(),rc(21,`a`,12)(22,`mat-icon`,13),PC(23,`download`),vf(),PC(24,` Download `),vf(),rc(25,`a`,14)(26,`mat-icon`,13),PC(27,`open_in_new`),vf(),PC(28,` GitHub `),vf()()(),rc(29,`mat-tab-nav-panel`,15,1),Fi(31,hn,1,1,`app-spec-source`,16)(32,gn,1,2,`app-redoc-view`,17),vf()),t&2){let a=mC(30);Mi(2),uv(`tabPanel`,a),Mi(),nC(e.apis),Mi(5),Dv(e.current().name),Mi(2),Dv(e.current().endpoint),Mi(3),nC(e.current().guides),Mi(3),uv(`value`,e.currentView()),Mi(5),uv(`href`,e.specUrl(),nb),Tn(`download`,e.current().file),Mi(4),uv(`href`,e.githubUrl(),nb),Mi(6),ji(e.currentView()===`source`?31:32)}},dependencies:[LG,Pt,Ft,qz,Ht,jt,Ie,Ct,ul,xt],styles:[`[_nghost-%COMP%]{display:block}.api-bar[_ngcontent-%COMP%]{position:sticky;inset-block-start:var(--%NS%app-header-height);z-index:5;background:var(--%NS%mat-sys-surface);border-block-end:1px solid var(--%NS%mat-sys-outline-variant)}.tabs[_ngcontent-%COMP%]{padding-inline:1rem}.spec-bar[_ngcontent-%COMP%]{display:flex;flex-wrap:wrap;align-items:center;gap:.5rem 1rem;padding:.5rem 1rem}.spec-title[_ngcontent-%COMP%]{display:flex;align-items:baseline;gap:.75rem}.spec-title[_ngcontent-%COMP%]   h1[_ngcontent-%COMP%]{margin:0;font:var(--%NS%mat-sys-title-large)}.guides[_ngcontent-%COMP%]{margin:0;color:var(--%NS%mat-sys-on-surface-variant)}.spacer[_ngcontent-%COMP%]{flex:1}.panel[_ngcontent-%COMP%]{display:block}@media(max-width:700px){.tabs[_ngcontent-%COMP%], .spec-bar[_ngcontent-%COMP%]{padding-inline:.5rem}.guides[_ngcontent-%COMP%], .github[_ngcontent-%COMP%], .spacer[_ngcontent-%COMP%]{display:none}}`]})};export{Ne as ApiReference};