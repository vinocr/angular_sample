'use strict';
angular.module('serviceCenter.router', [])
	.config(['$stateProvider', '$urlRouterProvider', function($stateProvider, $urlRouterProvider) {
    $urlRouterProvider.otherwise('/sc/dashboard');
    $stateProvider
        .state('sc', {
            url: '/sc',
            abstract: true,
            templateUrl: 'scripts/views/index.html',
            controller: 'serviceCenterController as scCtrl'
        })
        .state('sc.dashboard', {
            url: '/dashboard',
            views:{
                'base' :{
                    templateUrl: 'scripts/modules/dashboard/views/dashboard.html',
                    controller: 'dashboardController as dashboardCtrl'
                }
            }
        })
        .state('sc.allServices', {
            url: '/services',
            views:{
                'base' :{
                    templateUrl: 'scripts/modules/serviceCenter/views/servicesList.html',
                    controller: 'servicesListController as services'
                }
            }
        })
        .state('sc.allInstances', {
            url: '/instances',
            views:{
                'base' :{
                    templateUrl: 'scripts/modules/instances/views/instanceList.html',
                    controller: 'instancesListController as instances',
                }
            },
            resolve: {
                servicesList: ['$q', 'httpService', 'apiConstant',function($q, httpService, apiConstant){
                    $(".loader").show();
                    var deferred = $q.defer();
                    var url = apiConstant.api.microservice.url;
                    var method = apiConstant.api.microservice.method;
                    httpService.apiRequest(url,method).then(function(response){
                        $(".loader").hide();
                        if(response && response.data && response.data.services){
                            deferred.resolve(response.data.services);
                        }
                        else {
                            deferred.reject("no services");
                        }
                    },function(error){
                        $(".loader").hide();
                        deferred.reject(error);
                    });
                    return deferred.promise;
                }]
            }
        })
        .state('sc.info',{
            url: '/:serviceId',
            abstract: true,
            views: {
                'base': {
                    templateUrl: 'scripts/modules/serviceCenter/views/serviceInfo.html',
                    controller: 'serviceInfoController as serviceInfo'
                }
            },
            resolve: {
                serviceInfo: ['$q', 'httpService', 'commonService', 'apiConstant', '$stateParams', function($q, httpService, commonService, apiConstant, $stateParams){
                    $(".loader").show();
                    var serviceId = $stateParams.serviceId;
                    var deferred = $q.defer();
                    var url = apiConstant.api.microservice.url;
                    var method = apiConstant.api.microservice.method;
                    httpService.apiRequest(url,method).then(function(response){
                        $(".loader").hide();
                        if(response && response.data && response.data.services){
                            deferred.resolve(response);
                        }
                        else {
                            deferred.resolve(response);
                        }
                    },function(error){
                        $(".loader").hide();
                        deferred.reject(error);
                    });
                    return deferred.promise;
                }]
            }
        })
        .state('sc.info.instance', {
            url: '/instance',
            views: {
                "info" : {
                    templateUrl: 'scripts/modules/serviceCenter/views/serviceInstance.html'
                }
            }
        })
        .state('sc.info.provider', {
            url: '/provider',
            views: {
                "info" : {
                    templateUrl: 'scripts/modules/serviceCenter/views/serviceProvider.html'
                }
            }
        })
        .state('sc.info.consumer', {
            url: '/consumer',
            views: {
                "info" : {
                    templateUrl: 'scripts/modules/serviceCenter/views/serviceConsumer.html'
                }
            }
        })
        .state('sc.info.schema', {
            url: '/schema',
            views: {
                "info" : {
                    templateUrl: 'scripts/modules/serviceCenter/views/schema.html',
                    controller: 'schemaController as schemaCtrl'
                }
            },
            resolve: {
                servicesList: ['$q', 'httpService', 'apiConstant',function($q, httpService, apiConstant){
                    $(".loader").show();
                    var deferred = $q.defer();
                    var url = apiConstant.api.microservice.url;
                    var method = apiConstant.api.microservice.method;
                    httpService.apiRequest(url,method).then(function(response){
                        $(".loader").hide();
                        if(response && response.data && response.data.services){
                            deferred.resolve(response);
                        }
                        else {
                            deferred.resolve(response);
                        }
                    },function(error){
                        $(".loader").hide();
                        deferred.reject(error);
                    });
                    return deferred.promise;
                }]
            }
        })
        .state('error', {
        	url: '/error',
        	templateUrl: 'views/error.html'
        });
}]);
