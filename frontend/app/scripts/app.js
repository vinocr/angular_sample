//Copyright 2017 Huawei Technologies Co., Ltd
//
//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS,
//WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//See the License for the specific language governing permissions and
//limitations under the License.
'use strict';

angular.module('serviceCenter', ['ngAnimate', 'ngMaterial', 'ngAria', 'ngMessages', 'ngResource', 'ngRoute', 'ngSanitize', 'ui.router',
    'ngMdIcons', 'pascalprecht.translate', 'serviceCenter.router','md.data.table', 'jsonFormatter', 'chart.js'])
  .config(['$translateProvider', 'english', 'chinese', function($translateProvider, english, chinese) {
        $translateProvider.useSanitizeValueStrategy(null);
        
        $translateProvider.translations('en', english);
        $translateProvider.translations('cz', chinese);
  
        var lang = "";
        if(localStorage.getItem("lang") && localStorage.getItem("lang")!= ''){
            lang= localStorage.getItem("lang");
        }
        else if (navigator.language) {
            lang = navigator.language.indexOf("zh") > -1 ? "cz" : "en";
        } else {
            lang = navigator.userLanguage.indexOf("zh") > -1 ? "cz" : "en";
        }

        $translateProvider.preferredLanguage(lang);
    }])
  .config(['$httpProvider','$injector', function($httpProvider,$injector) {
        $httpProvider.defaults.useXDomain = true;
        delete $httpProvider.defaults.headers.common['X-Requested-With'];

        $injector.invoke(['$qProvider', function($qProvider) {
            $qProvider.errorOnUnhandledRejections(false);
        }]);
    }])
  .config(function (JSONFormatterConfigProvider) {
        JSONFormatterConfigProvider.hoverPreviewEnabled = true;
    });

